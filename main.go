package main

import (
	"context"

	"github.com/sirupsen/logrus"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/log"
	"github.com/osrg/gobgp/v3/pkg/server"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	//apb "google.golang.org/protobuf/types/known/anypb"
)

func main() {
	log := logrus.New()

	s := server.NewBgpServer(server.LoggerOption(&myLogger{logger: log}))
	go s.Serve()

	// global configuration
	if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			Asn:                   64513,
			RouterId:              "10.233.67.7",
			ListenPort:            1179, // gobgp won't listen on tcp:179
			RouteSelectionOptions: &api.RouteSelectionOptionsConfig{},
		},
	}); err != nil {
		log.Fatal(err)
	}

	var establishedChan = make(chan bool)

	// the change of the peer state and path
	if err := s.WatchEvent(context.Background(), &api.WatchEventRequest{
		Peer: &api.WatchEventRequest_Peer{},
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{

				{
					Type: api.WatchEventRequest_Table_Filter_BEST,
				},
				{
					Type: api.WatchEventRequest_Table_Filter_ADJIN,
				},
			},
		}}, func(r *api.WatchEventResponse) {
		if p := r.GetPeer(); p != nil && p.Type == api.WatchEventResponse_PeerEvent_STATE {
			log.Info(p)
			if p.Peer.State.SessionState == api.PeerState_ESTABLISHED {
				log.Info("established")
				establishedChan <- true
			}
		} else if t := r.GetTable(); t != nil {
			for _, p := range t.Paths {
				log.Info(t)
				prefix := &api.LabeledVPNIPAddressPrefix{}
				if err := anypb.UnmarshalTo(p.Nlri, prefix, proto.UnmarshalOptions{}); err != nil {
					continue
				}
				for _, pattr := range p.Pattrs {
					if pattr.TypeUrl == "type.googleapis.com/apipb.ExtendedCommunitiesAttribute" {
						extCommunitiesAttribute := &api.ExtendedCommunitiesAttribute{}
						if err := anypb.UnmarshalTo(pattr, extCommunitiesAttribute, proto.UnmarshalOptions{}); err != nil {
							log.Error(err)
						} else {
							for _, community := range extCommunitiesAttribute.Communities {
								switch community.TypeUrl {
								case "type.googleapis.com/apipb.TwoOctetAsSpecificExtended":
									extComm := &api.TwoOctetAsSpecificExtended{}
									if err := anypb.UnmarshalTo(community, extComm, proto.UnmarshalOptions{}); err != nil {
										log.Error(err)
									}
									if prefix.Prefix == "3.0.0.2" {
										log.Infof("prefix: %s\n", prefix.Prefix)
										log.Infof("community: target:%d:%d\n", extComm.Asn, extComm.LocalAdmin)
									}
									if extComm.Asn == 999999999 {
										log.Infof("prefix: %s\n", prefix.Prefix)
										log.Infof("community: target:%d:%d\n", extComm.Asn, extComm.LocalAdmin)
									}
								}
							}
						}

					}

				}

			}
		}
	}); err != nil {
		log.Fatal(err)
	}

	// neighbor configuration
	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: "10.233.67.2",
			PeerAsn:         64512,
			Type:            api.PeerType_EXTERNAL,
			LocalAsn:        64513,
		},
		ApplyPolicy: &api.ApplyPolicy{
			ImportPolicy: &api.PolicyAssignment{
				DefaultAction: api.RouteAction_ACCEPT,
			},
			ExportPolicy: &api.PolicyAssignment{
				DefaultAction: api.RouteAction_REJECT,
			},
		},
		AfiSafis: []*api.AfiSafi{
			{
				Config: &api.AfiSafiConfig{
					Family: &api.Family{
						Afi:  api.Family_AFI_IP,
						Safi: api.Family_SAFI_MPLS_VPN,
					},
					Enabled: true,
				},
			},
			{
				Config: &api.AfiSafiConfig{
					Family: &api.Family{
						Afi:  api.Family_AFI_IP,
						Safi: api.Family_SAFI_MPLS_LABEL,
					},
					Enabled: true,
				},
			},
		},
	}

	if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		log.Fatal(err)
	}

	if <-establishedChan {
		log.Info("continuing")
	}

	select {}
}

// implement github.com/osrg/gobgp/v3/pkg/log/Logger interface
type myLogger struct {
	logger *logrus.Logger
}

func (l *myLogger) Panic(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Panic(msg)
}

func (l *myLogger) Fatal(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Fatal(msg)
}

func (l *myLogger) Error(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Error(msg)
}

func (l *myLogger) Warn(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Warn(msg)
}

func (l *myLogger) Info(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Info(msg)
}

func (l *myLogger) Debug(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Debug(msg)
}

func (l *myLogger) SetLevel(level log.LogLevel) {
	l.logger.SetLevel(logrus.Level(level))
}

func (l *myLogger) GetLevel() log.LogLevel {
	return log.LogLevel(l.logger.GetLevel())
}

func getVRF(vrf *api.Vrf) {

}
