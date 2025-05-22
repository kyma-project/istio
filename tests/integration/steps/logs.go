package steps

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/avast/retry-go"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/kyma-project/istio/operator/tests/testcontext"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func ContainerLogContainsString(ctx context.Context, containerName, depName, depNamespace, expectedString string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	err = retry.Do(func() error {
		var dep v1.Deployment
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: depNamespace,
			Name:      depName,
		}, &dep)
		if err != nil {
			return err
		}

		var pods corev1.PodList
		err = k8sClient.List(ctx, &pods, client.MatchingLabels{
			"app": depName,
		})
		if err != nil {
			return err
		}

		var logStr = ""
		for _, pod := range pods.Items {
			logStr, err = getLogsFromPodsContainer(ctx, pod, containerName)
			if err != nil {
				return err
			}
			if sub := strings.Contains(logStr, expectedString); sub {
				return nil
			}
		}

		return fmt.Errorf("log entry not found. got log: %s", logStr)
	}, testcontext.GetRetryOpts()...)
	return ctx, err
}

func getLogsFromPodsContainer(ctx context.Context, pod corev1.Pod, containerName string) (string, error) {
	conf := config.GetConfigOrDie()
	c := kubernetes.NewForConfigOrDie(conf)

	logOpt := &corev1.PodLogOptions{
		Container: containerName,
	}
	req := c.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, logOpt)
	logs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		e := logs.Close()
		if e != nil {
			log.Printf("error closing logs stream: %s", err.Error())
		}
	}()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", err
	}
	str := buf.String()
	return str, nil
}
