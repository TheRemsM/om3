package daemondata

import "opensvc.com/opensvc/core/cluster"

// GetNodeStatus returns Monitor.Node.<node>
func GetNodeStatus(c chan<- any, node string) *cluster.TNode {
	result := make(chan *cluster.TNode)
	op := opGetNodeStatus{
		result: result,
		node:   node,
	}
	c <- op
	return <-result
}
