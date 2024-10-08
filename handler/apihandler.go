/*
Copyright 2024 Kubeworkz Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"kubeworkz-webconsole/errdef"
	"kubeworkz-webconsole/utils"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/patrickmn/go-cache"
	"github.com/saashqdev/kubeworkz/pkg/clog"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const PlatformKubeworkz = "kubeworkz"

func init() {
	clog.Info("webconsole initializing")

	initConfig()
	initAudit()

	clog.Info("webconsole initialized")
}

func CreateHTTPAPIHandler() http.Handler {
	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)

	apiV1Ws := new(restful.WebService)
	apiV1Ws.Filter(PodAuthorityVerify)

	apiV1Ws.Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	apiV2Ws := new(restful.WebService)

	apiV2Ws.Path("/api/v1/extends").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	wsContainer.Add(apiV1Ws)
	wsContainer.Add(apiV2Ws)

	apiV1Ws.Route(
		apiV1Ws.GET("{cluster}/namespace/{namespace}/pod/{pod}/shell/{container}").
			To(handleExecShell).
			Writes(TerminalResponse{}))
	//apiV1Ws.Route(
	//	apiV1Ws.GET("{cluster}/pod/{namespace}/{pod}/shell/{container}").
	//		To(handleExecShell).
	//		Writes(TerminalResponse{}))
	apiV2Ws.Route(
		apiV2Ws.GET("cloudShell/clusters/{cluster}").
			To(handleCloudShellExec).
			Writes(TerminalResponse{}))

	return wsContainer

}

// CreateAttachHandler is called from main for /api/sockjs
func CreateAttachHandler(path string) http.Handler {
	return sockjs.NewHandler(path, sockjs.DefaultOptions, handleTerminalSession)
}

// Handles execute shell API call
func handleExecShell(request *restful.Request, response *restful.Response) {

	sessionId, err := utils.GenTerminalSessionId()
	if err != nil {
		clog.Error("generate session id failed. Error msg: " + err.Error())
		errdef.HandleInternalError(response, err)
		return
	}
	clog.Info("sessionId: %s", sessionId)

	clusterName := request.PathParameter("cluster")

	// get restClient from map base on clusterName
	_, err = getNonControlCfg(clusterName)
	if err != nil {
		clog.Error("fail to fetch rest.config for cluster [%s], msg: %v", clusterName, err)
		errdef.HandleInternalErrorByCode(response, errdef.ClusterInfoNotFound)
		return
	}

	cInfo, errInfo := getConnInfo(request)
	if errInfo != nil {
		errdef.HandleInternalErrorByCode(response, *errInfo)
		return
	}
	cacheConnInfo(sessionId, cInfo)

	_ = response.WriteHeaderAndEntity(http.StatusOK, TerminalResponse{Id: sessionId})
}

func cacheConnInfo(sessionId string, info *ConnInfo) {
	v, _ := json.Marshal(info)
	// save container-connect info to sync.Map
	connMap.Store(sessionId, string(v))
}

func getConnInfo(request *restful.Request) (*ConnInfo, *errdef.ErrorInfo) {
	clusterName := request.PathParameter("cluster")
	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")
	containerName := request.PathParameter("container")

	scriptUser := request.QueryParameter("user")
	scriptUID := request.QueryParameter("uid")
	scriptUserAuth := request.QueryParameter("auth")

	// Audit related info
	webUser := request.QueryParameter("webuser")
	platform := request.QueryParameter("platform")
	// 未传入platform信息时，认为是Kubeworkz页面传入的
	if platform == "" {
		platform = PlatformKubeworkz
	}

	remoteIP := request.QueryParameter("remote_ip")
	if remoteIP == "" {
		remoteIP = request.HeaderParameter("X-Forwarded-For")
	}

	ua := request.QueryParameter("user_agent")
	if ua == "" {
		ua = request.HeaderParameter("User-Agent")
	}

	return &ConnInfo{
		Namespace:      namespace,
		PodName:        podName,
		ContainerName:  containerName,
		ClusterName:    clusterName,
		ScriptUID:      scriptUID,
		ScriptUser:     scriptUser,
		ScriptUserAuth: scriptUserAuth,
		AuditRawInfo: &AuditRawInfo{
			RemoteIP:  remoteIP,
			UserAgent: ua,
			WebUser:   webUser,
			Platform:  platform,
		},
	}, nil
}

// init rest.Config base on kubeconfig
func initKubeConf(kubeConfData string) *rest.Config {
	var err error
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfData))
	if err != nil {
		clog.Error("init kubeconfig failed, error msg: %s", err.Error())
		return nil
	}
	groupVersion := schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}
	cfg.GroupVersion = &groupVersion
	cfg.APIPath = "/api"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = scheme.Codecs
	return cfg
}

// get cfg from cache, if it is not in the cache, get it from K8s and update cache
func getNonControlCfg(clusterName string) (cfg *rest.Config, err error) {
	v, ok := configMap.Get(clusterName)
	if ok {
		return v.(*rest.Config), nil
	}
	// get cfg from k8s
	clog.Info("cluster [%s] config expire ot not exist in cache, try to fetch from K8s", clusterName)
	ci, err := GetClusterInfoByName(clusterName)
	if err != nil {
		return nil, err
	}
	data := string(ci.Spec.KubeConfig)
	// init rest client config, put it to cache
	NCfg := initKubeConf(data)
	if NCfg != nil {
		clog.Info("init rest client for cluster [%s] from config from K8s success", clusterName)
		configMap.Set(clusterName, NCfg, cache.DefaultExpiration)
	} else {
		msg := fmt.Sprintf("init rest client for cluster [%s] from config from db Fail, config data: %v", clusterName, data)
		clog.Error(msg)
		return nil, errors.New(msg)
	}
	return NCfg, nil
}
