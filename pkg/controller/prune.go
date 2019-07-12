package controller

import (
	"fmt"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

type pruneFunc func(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress

// FilterIngress filters ingress list based on filter functions and returns a filtered ingress list
func (c AppGwIngressController) PruneIngress(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) []*v1beta1.Ingress {

	pruneFuncList := []pruneFunc {
		c.pruneNoPrivateIP,
	}

	if cbCtx.EnvVariables.EnableBrownfieldDeployment == "true" {
		pruneFuncList  = append(pruneFuncList, c.pruneProhibitedIngress)
	}

	prunedIngressListed := cbCtx.IngressList
	for _, prune := range pruneFuncList {
		prunedIngressListed   = prune(appGw, cbCtx, prunedIngressListed)
	}

	return prunedIngressListed
}

func (c AppGwIngressController) pruneProhibitedIngress(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress {
	// Mutate the list of Ingresses by removing ones that AGIC should not be creating configuration.
	for idx, ingress := range ingressList {
		glog.V(5).Infof("Original Ingress[%d] Rules: %+v", idx, ingress.Spec.Rules)
		ingressList[idx].Spec.Rules = brownfield.PruneIngressRules(ingress, cbCtx.ProhibitedTargets)
		glog.V(5).Infof("Sanitized Ingress[%d] Rules: %+v", idx, ingress.Spec.Rules)
	}

	return ingressList
}

func (c AppGwIngressController) pruneNoPrivateIP(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress {
	pruneIngressList := make([]*v1beta1.Ingress, 0)
	appGwHasPrivateIP := appgw.GetIPConfigurationID(appGw, true) != nil
	for _, ingress := range ingressList {
		usePrivateIP, _ := annotations.UsePrivateIP(ingress)
		usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP == "true"
		if usePrivateIP && !appGwHasPrivateIP {
			errorLine := fmt.Sprintf("Removing Ingress with use Private IP annotation as Application Gateway doesn't have a private IP: %+v", ingress)
			glog.Error(errorLine)
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonNoPrivateIPError, errorLine)
		} else {
			pruneIngressList  = append(pruneIngressList , ingress)
		}
	}

	return pruneIngressList
}