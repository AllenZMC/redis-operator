package util

import (
	"fmt"
	"github.com/jw-s/redis-operator/pkg/operator"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/jw-s/redis-operator/pkg/operator/spec"
	"github.com/sirupsen/logrus"
	v1lister "k8s.io/client-go/listers/core/v1"
)

const (
	redisPort    = "6379"
	sentinelPort = "26379"
)

func GetMasterIPByName(client *redis.Client, name string) (string, error) {

	cmd := redis.NewStringSliceCmd("SENTINEL", "get-master-addr-by-name", name)

	err := client.Process(cmd)

	if err != nil {
		return "", err
	}

	masterAddr, err := cmd.Result()

	if err != nil {
		return "", err
	}

	operator.Logger.WithFields(logrus.Fields{"master_ip": masterAddr[0],
		"master_port": masterAddr[1]}).
		Debug("Master IP reported from sentinel(s)")

	return masterAddr[0], err
}

func GetSeedMasterIP(podLister v1lister.PodLister, namespace, name string) (string, error) {

	var masterIP string

	if err := WaitForResourceToBeEstablished(10, func() (bool, error) {
		masterSeed, err := podLister.Pods(namespace).Get(spec.GetMasterPodName(name))

		if err != nil {
			if ResourceNotFoundError(err) {
				return false, nil
			}

			return false, err
		}

		if IsPodReady(masterSeed) && masterSeed.Status.PodIP != "" {
			masterIP = masterSeed.Status.PodIP
			return true, nil
		}

		return false, nil

	}); err != nil {
		return "", err
	}

	operator.Logger.WithField("master_ip", masterIP).
		Debug("Got seed master IP")

	return masterIP, nil
}

func GetSlaveCount(client *redis.Client, name string) int {
	var count int
	cmd := redis.NewSliceCmd("SENTINEL", "slaves", name)

	err := client.Process(cmd)

	if err != nil {
		operator.Logger.Error(err.Error())
		return count
	}

	result, err := cmd.Result()

	if err != nil {
		operator.Logger.Error(err.Error())
		return count
	}

	for _, slaveBlob := range result {
		if _, ok := slaveBlob.([]interface{}); ok {
			count++
		}
	}

	operator.Logger.WithField("count", count).
		Debug("slaves avaliable")

	return count
}

func NewSentinelRedisClient(name string) *redis.Client {
	sentinelService := fmt.Sprintf("%s:%v", name, spec.RedisSentinelPort)

	return redis.NewClient(&redis.Options{
		Addr:            sentinelService,
		Password:        "",
		DB:              0,
		MaxRetries:      10,
		DialTimeout:     time.Second * 30,
		MinRetryBackoff: time.Second * 30,
	})
}

//哨兵
func SetCustomSentinelConfig(ip string, configs []string, masterName string) error {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", ip, sentinelPort),
		Password: "",
		DB:       0,
	}
	rClient := redis.NewClient(options)
	defer rClient.Close()

	for _, config := range configs {
		param, value, err := getConfigParameters(config)
		if err != nil {
			return err
		}
		if err := applySentinelConfig(param, value, masterName, rClient); err != nil {
			return err
		}
	}
	return nil
}

func applySentinelConfig(parameter, value, masterName string, rClient *redis.Client) error {
	cmd := redis.NewStatusCmd("SENTINEL", "set", masterName, parameter, value)
	rClient.Process(cmd)
	return cmd.Err()
}

//redis
func SetCustomRedisConfig(ip string, configs []string) error {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", ip, redisPort),
		Password: "",
		DB:       0,
	}
	rClient := redis.NewClient(options)
	defer rClient.Close()

	for _, config := range configs {
		param, value, err := getConfigParameters(config)
		if err != nil {
			return err
		}
		if err := applyRedisConfig(param, value, rClient); err != nil {
			return err
		}
		result := rClient.ConfigGet(param)
		operator.Logger.Debug(result)
	}
	return nil
}

func applyRedisConfig(parameter string, value string, rClient *redis.Client) error {
	result := rClient.ConfigSet(parameter, value)
	operator.Logger.Error("applyRedisConfig: ", result.Err())
	return result.Err()
}

func getConfigParameters(config string) (parameter string, value string, err error) {
	s := strings.Split(config, " ")
	operator.Logger.Debug(s)

	if len(s) < 2 {
		return "", "", fmt.Errorf("configuration '%s' malformed", config)
	}
	operator.Logger.Debug(s[0], strings.Join(s[1:], " "))

	return s[0], strings.Join(s[1:], " "), nil
}
