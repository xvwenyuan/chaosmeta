/*
 * Copyright 2022-2023 Chaos Meta Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cloudnativeexecutor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/api/v1alpha1"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/pkg/model"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/pkg/restclient"
	corev1 "k8s.io/api/core/v1"
	"time"
)

func init() {
	registerCloudExecutor(v1alpha1.PodCloudTarget, "finalizer", &PodFinalizerExecutor{})
}

type PodFinalizerExecutor struct{}

func (e *PodFinalizerExecutor) Inject(ctx context.Context, injectObject, uid, timeout string, args []v1alpha1.ArgsUnit) (string, error) {
	ns, name, _, err := model.ParsePodInfo(injectObject)
	if err != nil {
		return "", fmt.Errorf("unexpected pod format: %s", err.Error())
	}

	c := restclient.GetApiServerClientMap(v1alpha1.PodCloudTarget)
	pod := &corev1.Pod{}
	if err := c.Get().Namespace(ns).Resource("pods").Name(name).Do(ctx).Into(pod); err != nil {
		return "", fmt.Errorf("get pod error: %s", err.Error())
	}

	var backupBytes []byte
	if pod.ObjectMeta.Finalizers != nil {
		backupBytes, err = json.Marshal(pod.ObjectMeta.Finalizers)
		if err != nil {
			return "", fmt.Errorf("backup to string error: %s", err.Error())
		}
	}

	return string(backupBytes), patchFinalizers(ctx, c, "pods", ns, name, getNewFinalizers(ctx, pod.ObjectMeta.Finalizers, args))
}

func (e *PodFinalizerExecutor) Recover(ctx context.Context, injectObject, uid, backup string) error {
	ns, name, _, err := model.ParsePodInfo(injectObject)
	if err != nil {
		return fmt.Errorf("unexpected pod format: %s", err.Error())
	}

	var oldFinalizers []string
	if backup != "" {
		if err := json.Unmarshal([]byte(backup), &oldFinalizers); err != nil {
			return fmt.Errorf("get old finalizers error: %s", err.Error())
		}
	}

	c := restclient.GetApiServerClientMap(v1alpha1.PodCloudTarget)
	return patchFinalizers(ctx, c, "pods", ns, name, oldFinalizers)
}

func (e *PodFinalizerExecutor) Query(ctx context.Context, injectObject, uid, backup string, phase v1alpha1.PhaseType) (*model.SubExpInfo, error) {
	return &model.SubExpInfo{
		UID:        uid,
		Status:     v1alpha1.SuccessStatusType,
		UpdateTime: time.Now().Format(model.TimeFormat),
	}, nil
}
