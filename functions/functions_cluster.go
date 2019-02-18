package functions

type ClusterFunctions struct {
	global *GlobalClusterFunctions
	local  *LocalClusterFunctions
}

func (f *ClusterFunctions) Global() *GlobalClusterFunctions {
	return f.global
}

func (f *ClusterFunctions) Local() *LocalClusterFunctions {
	return f.local
}
