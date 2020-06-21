package spec

const (
	RedisPort                      = 6379
	RedisSentinelPort              = 26379
	MasterPodName                  = "redis-master-%s"
	MasterServiceName              = "redis-master-%s"
	SentinelServiceName            = "redis-sentinel-%s"
	SentinelDeploymentName         = "redis-sentinel-%s"
	SlaveStatefulSetName           = "redis-slave-%s"
	SlavePersistentVolumeClaimName = "slave-persistent-storage"
	OperatorLabel                  = "redis_operator"
	SentinelConfigMapName          = "sentinel-config-%s"
	RedisConfigMapName             = "redis-config-%s"
	ConfigVolumeName               = "config"
	DataVolumeName                 = "data"
	sentinelConfigFileName         = "sentinel.conf"
	redisConfigFileName            = "redis.conf"
	ConfMountPath                  = "/usr/local/etc/redis"
	DataMountPath                  = "/data"
)
