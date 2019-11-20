package node

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"

	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	nodePeersGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: "p2p_peers_count",
		Help: "Current numbers of peers split by name.",
	}, []string{"type", "version", "platform"})
	nodePeersChanges = prom.NewGauge(prom.GaugeOpts{
		Name: "p2p_peers_changes",
		Help: "Current changes to connected peers.",
	})
	nodePeersAbsolute = prom.NewGauge(prom.GaugeOpts{
		Name: "p2p_peers_absolute",
		Help: "Absolute number of connected peers.",
	})
	nodeMaxPeersGauge = prom.NewGauge(prom.GaugeOpts{
		Name: "p2p_peers_max",
		Help: "Maximum number of peers that can connect.",
	})
)

func init() {
	prom.MustRegister(nodePeersGauge)
	prom.MustRegister(nodePeersChanges)
	prom.MustRegister(nodePeersAbsolute)
	prom.MustRegister(nodeMaxPeersGauge)
}

func updateNodeMetrics(node *node.Node, evType p2p.PeerEventType) error {
	server := node.Server()
	if server == nil {
		return errors.New("p2p server is unavailable")
	}

	calculatePeerCounts(server)

	change, err := computeMetrics(server, evType)
	if err != nil {
		return err
	}

	nodePeersChanges.Add(change.Counter)
	nodePeersAbsolute.Set(change.Absolute)
	nodeMaxPeersGauge.Set(change.Max)

	return nil
}

type peersChange struct {
	Counter  float64
	Absolute float64
	Max      float64
}

func labelsFromNodeName(name string) (prom.Labels, error) {
	tokens := strings.Split(name, "/")
	if len(tokens) == 4 {
		return prom.Labels{
			"type":     tokens[0],
			"version":  tokens[1],
			"platform": tokens[2],
		}, nil
	} else if len(tokens) == 3 {
		return prom.Labels{
			"type":     tokens[0],
			"version":  "unknown",
			"platform": tokens[1],
		}, nil
	} else {
		return nil, errors.New("wrong number of segments in name")
	}
}

func calculatePeerCounts(server *p2p.Server) {
	peers := server.Peers()
	/* necessary to count all peers anew */
	nodePeersGauge.Reset()

	for _, p := range peers {
		labels, err := labelsFromNodeName(p.Name())
		if err != nil {
			logger.Warn("failed parsing peer name", "error", err, "name", p.Name())
			continue
		}
		nodePeersGauge.With(labels).Inc()
	}
}

func computeMetrics(server *p2p.Server, evType p2p.PeerEventType) (result peersChange, err error) {
	switch evType {
	case p2p.PeerEventTypeAdd:
		result.Counter = 1
	case p2p.PeerEventTypeDrop:
		result.Counter = -1
	}

	result.Absolute = float64(server.PeerCount())
	result.Max = float64(server.MaxPeers)
	return
}
