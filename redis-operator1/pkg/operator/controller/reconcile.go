package controller

import (
	"encoding/json"
	"fmt"
	"github.com/jw-s/redis-operator/pkg/operator"
	"reflect"
	"strings"

	redisv1 "github.com/jw-s/redis-operator/pkg/apis/redis/v1"
	"github.com/jw-s/redis-operator/pkg/errors"
	"github.com/jw-s/redis-operator/pkg/operator/redis"
	"github.com/jw-s/redis-operator/pkg/operator/spec"
	"github.com/jw-s/redis-operator/pkg/operator/util"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func (c *RedisController) reconcile(redis *redis.Redis) error {

	var masterIP string

	if !redis.SeedMasterProcessComplete {
		if err := c.seedMasterPodProcess(redis.Redis); err != nil {
			return err
		}

		if err := redis.MarkAddSeedMasterCondition(); err != nil {
			return err
		}

		seedIP, err := util.GetSeedMasterIP(c.podLister, redis.Redis.Namespace, redis.Redis.Name)

		if err != nil {
			return err
		}

		masterIP = seedIP

		redis.SeedMasterProcessComplete = true

	} else { //由于resync机制，所以默认一分钟后会触发更新事件，进入此分支
		//先删除的话，哨兵可能获取不到master ip，这个时候master就是空，哨兵就会与redis集群断开了，不符合需求
		//所以先获取master ip，再删除，等待哨兵选举出来新的master
		ip, err := util.GetMasterIPByName(redis.Config.RedisClient, spec.GetRedisMasterName(redis.Redis))

		if err != nil || (util.GetSlaveCount(redis.Config.RedisClient, spec.GetRedisMasterName(redis.Redis)) == 0) {
			// Something went wrong, mark to spin up seed pod on next run
			redis.SeedMasterProcessComplete = false //重新创建种子节点0
			operator.Logger.Error("GetMasterIPByName err: ", err)
			return err
		}

		masterIP = ip

		deletePolicy := v1.DeletePropagationForeground
		//干掉seed master，由哨兵选举某个slave为新的master
		err = c.kubernetesClient.CoreV1().Pods(redis.Redis.Namespace).Delete(spec.GetMasterPodName(redis.Redis.Name),
			&metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
		//master pod 没有删除掉，是其他类型的错误
		if err != nil && !util.ResourceNotFoundError(err) { //不管删除还是查找，如果没有资源，都会报资源不存在错误
			redis.SeedMasterDeleted = false
			return err
		}
		//master pod 删除了
		if !redis.SeedMasterDeleted {
			if err := redis.MarkRemoveSeedMasterCondition(); err != nil {
				return err
			}
		}

		redis.SeedMasterDeleted = true
	}

	if err := c.masterEndpointProcess(redis.Redis, masterIP); err != nil {
		return err
	}

	if err := c.masterServiceProcess(redis.Redis); err != nil {
		return err
	}

	if err := c.sentinelConfigProcess(redis.Redis); err != nil {
		return err
	}

	if err := c.sentinelServiceProcess(redis.Redis); err != nil {
		return err
	}

	if err := c.sentinelProcess(redis.Redis); err != nil {
		return err
	}

	if err := c.redisConfigProcess(redis.Redis); err != nil {
		return err
	}

	return c.slaveProcess(redis.Redis)

}
func (c *RedisController) seedMasterPodProcess(redis *redisv1.Redis) error {

	_, err := c.podLister.Pods(redis.Namespace).Get(spec.GetMasterPodName(redis.Name))

	if err != nil {
		if util.ResourceNotFoundError(err) {
			if _, err := c.kubernetesClient.CoreV1().Pods(redis.Namespace).Create(spec.MasterSeedPod(redis)); err != nil {
				return err
			}

			return nil
		}

		return err
	}

	return nil

}

func (c *RedisController) sentinelProcess(redis *redisv1.Redis) error {

	actual, err := c.deploymentLister.Deployments(redis.Namespace).Get(spec.GetSentinelDeploymentName(redis.Name))

	if err != nil {
		if util.ResourceNotFoundError(err) {

			return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.SentinelDeployment(redis))
		}

		return err
	}

	return c.updateToDesired(redis.Namespace, actual.DeepCopy(), spec.SentinelDeployment(redis))

}

func (c *RedisController) sentinelConfigProcess(redis *redisv1.Redis) error {

	configMapName := string(redis.Spec.Sentinels.ConfigMap)

	_, err := c.configMapLister.ConfigMaps(redis.Namespace).Get(configMapName)

	if err != nil {
		if util.ResourceNotFoundError(err) {

			return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.DefaultSentinelConfig(redis))
		}

		return err
	}
	//更新哨兵配置（使用sentinel set mymaster）
	sentinels, err := util.GetIPs(c.GetDeploymentPods, redis, spec.GetSentinelDeploymentName(redis.Name))
	if err != nil {
		return err
	}
	for _, sip := range sentinels {
		if err := util.SetCustomSentinelConfig(sip, redis.Spec.Sentinels.CustomConfig, redis.Name); err != nil {
			return err
		}
	}

	return nil
}

func (c *RedisController) GetDeploymentPods(namespace, name string) (*apiv1.PodList, error) {
	deployment, err := c.deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return c.GetPods(deployment.Spec.Selector.MatchLabels, namespace)
}

func (c *RedisController) GetPods(matchLabels map[string]string, namespace string) (*apiv1.PodList, error) {
	labels := make([]string, 0)
	for k, v := range matchLabels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	selector := strings.Join(labels, ",")
	return c.kubernetesClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector})
}

func (c *RedisController) GetStatefulSetPods(namespace, name string) (*apiv1.PodList, error) {
	statefulSet, err := c.statefulSetLister.StatefulSets(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return c.GetPods(statefulSet.Spec.Selector.MatchLabels, namespace)
}

func (c *RedisController) redisConfigProcess(redis *redisv1.Redis) error {
	//只为了第一次创建operator时，跳过更新config操作
	configMapName := string(redis.Spec.Slaves.ConfigMap)
	_, err := c.configMapLister.ConfigMaps(redis.Namespace).Get(configMapName)

	if err != nil {
		if util.ResourceNotFoundError(err) {
			return nil
			//return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.DefaultRedisConfig(redis))
		}
		return err
	}

	//更新redis配置（使用config set）
	redises, err := util.GetIPs(c.GetStatefulSetPods, redis, spec.GetSlaveStatefulSetName(redis.Name))
	if err != nil {
		return err
	}
	for _, rip := range redises {
		if err := util.SetCustomRedisConfig(rip, redis.Spec.Slaves.CustomConfig); err != nil {
			return err
		}
	}

	return nil
}

func (c *RedisController) slaveProcess(redis *redisv1.Redis) error {

	actual, err := c.statefulSetLister.StatefulSets(redis.Namespace).Get(spec.GetSlaveStatefulSetName(redis.Name))

	if err != nil {
		if util.ResourceNotFoundError(err) {

			return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.SlaveStatefulSet(redis))
		}

		return err
	}

	return c.updateToDesired(redis.Namespace, actual.DeepCopy(), spec.SlaveStatefulSet(redis))

}

func (c *RedisController) sentinelServiceProcess(redis *redisv1.Redis) error {

	actual, err := c.serviceLister.Services(redis.Namespace).Get(spec.GetSentinelServiceName(redis.Name))

	if err != nil {
		if util.ResourceNotFoundError(err) {

			return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.SentinelService(redis))

		}

		return err
	}

	return c.updateToDesired(redis.Namespace, actual.DeepCopy(), spec.SentinelService(redis))

}

func (c *RedisController) masterServiceProcess(redis *redisv1.Redis) error {

	actual, err := c.serviceLister.Services(redis.Namespace).Get(spec.GetMasterServiceName(redis.Name))

	if err != nil {
		if util.ResourceNotFoundError(err) {

			return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.MasterService(redis))
		}

		return err
	}

	return c.updateToDesired(redis.Namespace, actual.DeepCopy(), spec.MasterService(redis))
}

func (c *RedisController) masterEndpointProcess(redis *redisv1.Redis, ipAddress string) error {

	actual, err := c.endpointsLister.Endpoints(redis.Namespace).Get(spec.GetMasterServiceName(redis.Name))

	if err != nil {
		if util.ResourceNotFoundError(err) {

			return util.CreateKubeResource(c.kubernetesClient, redis.Namespace, spec.MasterServiceEndpoint(redis, ipAddress))
		}

		return err
	}

	return c.updateToDesired(redis.Namespace, actual.DeepCopy(), spec.MasterServiceEndpoint(redis, ipAddress))

}

func (c *RedisController) deleteResources(namespace, name string) error {

	deletePolicy := metav1.DeletePropagationBackground

	err := c.kubernetesClient.CoreV1().Pods(namespace).Delete(spec.GetMasterPodName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	err = c.kubernetesClient.AppsV1beta1().Deployments(namespace).Delete(spec.GetSlaveStatefulSetName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	err = c.kubernetesClient.AppsV1beta1().Deployments(namespace).Delete(spec.GetSentinelDeploymentName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	err = c.kubernetesClient.CoreV1().Services(namespace).Delete(spec.GetSentinelServiceName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	err = c.kubernetesClient.CoreV1().ConfigMaps(namespace).Delete(spec.GetSentinelConfigMapName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	err = c.kubernetesClient.AppsV1beta1().StatefulSets(namespace).Delete(spec.GetSlaveStatefulSetName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	err = c.kubernetesClient.CoreV1().Services(namespace).Delete(spec.GetMasterServiceName(name), &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil && !util.ResourceNotFoundError(err) {
		return err
	}

	return nil
}

func (c *RedisController) updateToDesired(namespace string, actual interface{}, desired interface{}) error {

	c.logger.Debug("Updating resource")
	defer c.logger.Debug("Finished updating resource")

	switch t := actual.(type) {

	case *appsv1beta1.Deployment:

		patch, err := createStrategicMergePatch(actual, desired, appsv1beta1.Deployment{})

		if err != nil {
			return err
		}

		_, err = c.kubernetesClient.AppsV1beta1().Deployments(namespace).Patch(t.Name, types.StrategicMergePatchType, patch)

		return err

	case *appsv1beta1.StatefulSet:

		patch, err := createStrategicMergePatch(actual, desired, appsv1beta1.StatefulSet{})

		if err != nil {
			return err
		}
		_, err = c.kubernetesClient.AppsV1beta1().StatefulSets(namespace).Patch(t.Name, types.StrategicMergePatchType, patch)

		return err

	case *apiv1.Service:

		if !reflect.DeepEqual(actual, desired) {

			desiredJSON, err := json.Marshal(desired)
			if err != nil {
				return err
			}

			_, err = c.kubernetesClient.CoreV1().Services(namespace).Patch(t.Name, types.StrategicMergePatchType, desiredJSON)

			return err
		}

		return nil

	case *apiv1.Endpoints:

		patch, err := createStrategicMergePatch(actual, desired, apiv1.Endpoints{})

		if err != nil {
			return err
		}
		_, err = c.kubernetesClient.CoreV1().Endpoints(namespace).Patch(t.Name, types.StrategicMergePatchType, patch)

		return err

	case *apiv1.ConfigMap:

		patch, err := createStrategicMergePatch(actual, desired, apiv1.ConfigMap{})

		if err != nil {
			return err
		}
		_, err = c.kubernetesClient.CoreV1().ConfigMaps(namespace).Patch(t.Name, types.StrategicMergePatchType, patch)

		return err

	default:
		return errors.UnsupportedKubeResource
	}

}

func createStrategicMergePatch(actual interface{}, desired interface{}, dataStruct interface{}) ([]byte, error) {

	if !reflect.DeepEqual(actual, desired) {

		actualJSON, err := json.Marshal(actual)

		if err != nil {
			return nil, err
		}
		desiredJSON, err := json.Marshal(desired)
		if err != nil {
			return nil, err
		}

		patch, err := strategicpatch.CreateTwoWayMergePatch(actualJSON, desiredJSON, dataStruct)

		if err != nil {
			return nil, err
		}

		return patch, nil
	}

	return nil, nil
}
