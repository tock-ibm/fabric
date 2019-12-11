/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package smartbft

import "github.com/hyperledger/fabric/common/metrics"

var (
	clusterSizeOpts = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "smartbft",
		Name:         "cluster_size",
		Help:         "Number of nodes in this channel.",
		LabelNames:   []string{"channel"},
		StatsdFormat: "%{#fqname}.%{channel}",
	}
	committedBlockNumberOpts = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "smartbft",
		Name:         "committed_block_number",
		Help:         "The number of the latest committed block.",
		LabelNames:   []string{"channel"},
		StatsdFormat: "%{#fqname}.%{channel}",
	}
)

type Metrics struct {
	ClusterSize          metrics.Gauge
	CommittedBlockNumber metrics.Gauge
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		ClusterSize:          p.NewGauge(clusterSizeOpts),
		CommittedBlockNumber: p.NewGauge(committedBlockNumberOpts),
	}
}
