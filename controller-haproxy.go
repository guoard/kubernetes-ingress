package main

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/haproxytech/models"
)

func (c *HAProxyController) updateHAProxy(reloadRequested bool) error {
	nativeAPI := c.NativeAPI

	c.handleDefaultTimeouts()
	version, err := nativeAPI.Configuration.GetVersion("")
	if err != nil || version < 1 {
		//silently fallback to 1
		version = 1
	}
	//log.Println("Config version:", version)
	transaction, err := nativeAPI.Configuration.StartTransaction(version)
	c.ActiveTransaction = transaction.ID
	defer func() {
		c.ActiveTransaction = ""
	}()
	if err != nil {
		log.Println(err)
		return err
	}

	if maxconnAnn, err := GetValueFromAnnotations("maxconn", c.cfg.ConfigMap.Annotations); err == nil {
		if maxconn, err := strconv.ParseInt(maxconnAnn.Value, 10, 64); err == nil {
			if maxconnAnn.Status == DELETED {
				maxconnAnn, _ = GetValueFromAnnotations("maxconn", c.cfg.ConfigMap.Annotations) // has default
				maxconn, _ = strconv.ParseInt(maxconnAnn.Value, 10, 64)
			}
			if maxconnAnn.Status != "" {
				err := c.handleMaxconn(transaction, maxconn, FrontendHTTP, FrontendHTTPS)
				if err != nil {
					return err
				}
			}
		}
	}

	maxProcs, maxThreads, reload, err := c.handleGlobalAnnotations(transaction)
	LogErr(err)
	reloadRequested = reloadRequested || reload
	pathIndex := 0

	var usingHTTPS bool
	reload, usingHTTPS, err = c.handleHTTPS(maxProcs, maxThreads, transaction)
	if err != nil {
		return err
	}
	err = c.handleRateLimiting(transaction, usingHTTPS)
	if err != nil {
		return err
	}
	numProcs, _ := strconv.Atoi(maxProcs.Value)
	numThreads, _ := strconv.Atoi(maxThreads.Value)
	port := int64(80)
	listener := &models.Bind{
		Name:    "http_1",
		Address: "0.0.0.0",
		Port:    &port,
		Process: "1/1",
	}
	if !usingHTTPS {
		if numProcs > 1 {
			listener.Process = "all"
		}
		if numThreads > 1 {
			listener.Process = "all"
		}
	}
	if listener.Process != c.cfg.HTTPBindProcess {
		if err = nativeAPI.Configuration.EditBind(listener.Name, FrontendHTTP, listener, transaction.ID, 0); err != nil {
			return err
		}
		c.cfg.HTTPBindProcess = listener.Process
	}
	reloadRequested = reloadRequested || reload
	reload, err = c.handleHTTPRedirect(usingHTTPS, transaction)
	if err != nil {
		return err
	}
	reloadRequested = reloadRequested || reload

	backendsUsed := map[string]int{}
	for _, namespace := range c.cfg.Namespace {
		if !namespace.Relevant {
			continue
		}
		//TODO, do not just go through them, sort them to handle /web,/ maybe?
		for _, ingress := range namespace.Ingresses {
			//no need for switch/case for now
			sortedList := make([]string, len(ingress.Rules))
			index := 0
			for name, _ := range ingress.Rules {
				sortedList[index] = name
				index++
			}
			sort.Strings(sortedList)
			for _, ruleName := range sortedList {
				rule := ingress.Rules[ruleName]
				indexedPaths := make([]*IngressPath, len(rule.Paths))
				for _, path := range rule.Paths {
					if path.Status != DELETED {
						indexedPaths[path.PathIndex] = path
					} else {
						delete(c.cfg.UseBackendRules, fmt.Sprintf("R%0006d", pathIndex))
						c.cfg.UseBackendRulesStatus = MODIFIED
						log.Println("SKIPPED", path)
					}
				}
				for i, _ := range indexedPaths {
					path := indexedPaths[i]
					if path == nil {
						continue
					}
					err := c.handlePath(pathIndex, namespace, ingress, rule, path, transaction, backendsUsed)
					LogErr(err)
					pathIndex++
				}
			}
		}
	}
	//handle default service
	c.handleDefaultService(transaction, backendsUsed)
	for backendName, numberOfTimesBackendUsed := range backendsUsed {
		if numberOfTimesBackendUsed < 1 {
			err := nativeAPI.Configuration.DeleteBackend(backendName, transaction.ID, 0)
			LogErr(err)
		}
	}

	LogErr(err)
	err = c.requestsTCPRefresh(transaction)
	LogErr(err)
	err = c.RequestsHTTPRefresh(transaction)
	LogErr(err)
	err = c.useBackendRuleRefresh()
	LogErr(err)
	_, err = nativeAPI.Configuration.CommitTransaction(transaction.ID)
	if err != nil {
		log.Println(err)
		return err
	}
	c.cfg.Clean()
	if reloadRequested {
		if err := c.HAProxyReload(); err != nil {
			log.Println(err)
		} else {
			log.Println("HAProxy reloaded")
		}
	} else {
		log.Println("HAProxy updated without reload")
	}
	return nil
}

func (c *HAProxyController) handleMaxconn(transaction *models.Transaction, maxconn int64, frontends ...string) error {
	for _, frontendName := range frontends {
		if _, frontend, err := c.NativeAPI.Configuration.GetFrontend(frontendName, transaction.ID); err == nil {
			frontend.Maxconn = &maxconn
			err := c.NativeAPI.Configuration.EditFrontend(frontendName, frontend, transaction.ID, 0)
			LogErr(err)
		} else {
			return err
		}
	}
	return nil
}

func (c *HAProxyController) handleDefaultService(transaction *models.Transaction, backendsUsed map[string]int) error {
	dsvcData, _ := GetValueFromAnnotations("default-backend-service")
	dsvc := strings.Split(dsvcData.Value, "/")

	if len(dsvc) != 2 {
		return errors.New("default service invalid data")
	}
	namespace, ok := c.cfg.Namespace[dsvc[0]]
	if !ok {
		return errors.New("default service invalid namespace " + dsvc[0])
	}
	ingress := &Ingress{
		Namespace:   namespace.Name,
		Annotations: MapStringW{},
		Rules:       map[string]*IngressRule{},
	}
	path := &IngressPath{
		ServiceName: dsvc[1],
		PathIndex:   -1,
	}
	return c.handlePath(0, namespace, ingress, nil, path, transaction, backendsUsed)
}