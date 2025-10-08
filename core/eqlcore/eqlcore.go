package eqlcore

import "C"

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"sync"
	// "time"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	//relay "github.com/libp2p/go-libp2p-circuit"
	// "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	multiaddr "github.com/multiformats/go-multiaddr"
	//"github.com/libp2p/go-libp2p/p2p/discovery"
)

// MessageReceiver interface for Kotlin callbacks
type MessageReceiver interface {
	OnMessageReceived(from string, msg string)
}

// глобальная нода
var (
	ctx             context.Context
	cancel          context.CancelFunc
	node            *chatNodeInternal
	messageReceiver MessageReceiver
	receiverMutex   sync.RWMutex
)

// публичная структура для Kotlin — только строки
type NodeInfo struct {
	ID    string
	Addrs []string
}

type topicStruct struct {
	Topic *pubsub.Topic
	Sub   *pubsub.Subscription
}
// инкапсулированная нода (не экспортируется в Kotlin)
type chatNodeInternal struct {
	Node     host.Host
	PubSub   *pubsub.PubSub
	Topics    map[string]*topicStruct
	MdnsService mdns.Service
}

type mdnsNotifee struct {
	host host.Host
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Println("mDNS обнаружил пира:", pi.ID)
	if err := n.host.Connect(context.Background(), pi); err != nil {
		log.Println("Ошибка подключения к пирам:", err)
	} else {
		log.Println("✅ Подключились к", pi.ID)
	}
}
// --- Вспомогательные функции ---

func createIdentity() (crypto.PrivKey, string) {
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Println("Ошибка генерации ключа:", err)
		return nil, "ERROR: createIdentity failed"
	}
	return priv, ""
}

func initDHT(ctx context.Context, h host.Host) *dht.IpfsDHT {
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		log.Println("Ошибка создания DHT:", err)
		return nil
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		log.Println("Ошибка bootstrap DHT:", err)
	}
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				log.Println("Bootstrap warning:", err)
			}
		}()
	}
	wg.Wait()
	return kademliaDHT
}

func discoverPeers(ctx context.Context, h host.Host, topic string) {
	kademliaDHT := initDHT(ctx, h)
	if kademliaDHT == nil {
		log.Println("DHT is nil, skipping peer discovery")
		return
	}

	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topic)

	peerChan, err := routingDiscovery.FindPeers(ctx, topic)
	if err != nil {
		log.Println("Error finding peers:", err)
		return
	}

	for p := range peerChan {
		if p.ID == h.ID() {
			continue
		}
		if err := h.Connect(ctx, p); err == nil {
			log.Println("Connected to:", p.ID)
		} else {
			log.Println("Failed to connect:", err)
		}
	}
}

// --- API для Kotlin ---
// Возвращаем строку: либо ID, либо "ERROR: ..."
func StartNode(listenAddr string, topic string) string {
	ctx, cancel = context.WithCancel(context.Background())

	addr, _ := multiaddr.NewMultiaddr(listenAddr)
	priv, errStr := createIdentity()
	if priv == nil {
		return errStr
	}

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrs(addr),
		// libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.EnableHolePunching(),
	)

	if err != nil {
		log.Println("Ошибка создания ноды:", err)
		return "ERROR: " + err.Error()
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Println("Ошибка создания PubSub:", err)
		return "ERROR: " + err.Error()
	}

	notifee := &mdnsNotifee{host: h}
	service := mdns.NewMdnsService(h, "", notifee)
	if err := service.Start(); err != nil {
		log.Println("Ошибка запуска mDNS:", err)
	}

	node = &chatNodeInternal{
		Node:   h,
		PubSub: ps,
		MdnsService: service,
	}

	SubscribeToTopic(topic)
	//go discoverPeers(ctx, h, topic)

	return h.ID().String()
}

func StopNode() string {
	if cancel != nil {
		cancel()
	}
	if node != nil {
		node.Node.Close()
		node.MdnsService.Close()
	}
	return "OK"
}

func SendMessage(msg string, topic string) string {
	if node == nil || node.Topics == nil {
		return "ERROR: node not initialized"
	}

	if _, ok := node.Topics[topic]; !ok {
		SubscribeToTopic(topic)
	}

	if err := node.Topics[topic].Topic.Publish(ctx, []byte(msg)); err != nil {
		log.Println("Ошибка отправки сообщения:", err)
		return "ERROR: " + err.Error()
	}
	return "OK"
}

func ConnectPeer(addr string) string {
	if node == nil {
		return "ERROR: node not initialized"
	}
	peerAddr, err := peer.AddrInfoFromString(addr)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if err := node.Node.Connect(ctx, *peerAddr); err != nil {
		return "ERROR: " + err.Error()
	}
	return "OK"
}

// SetMessageReceiver устанавливает receiver для callback-ов
func SetMessageReceiver(receiver MessageReceiver) {
	receiverMutex.Lock()
	defer receiverMutex.Unlock()
	messageReceiver = receiver
}

func SubscribeToTopic(topic string) string {
	if node == nil || node.PubSub == nil {
		return "node not initialized"
	}

	if _, ok := node.Topics[topic]; ok {
		return "already subscribed"
	}

	t, err := node.PubSub.Join(topic)
	if err != nil {
		log.Println("Ошибка подписки на топик:", err)
		return "ERROR: " + err.Error()
	}

	sub, err := t.Subscribe()
	if err != nil {
		log.Println("Ошибка подписки:", err)
		return "ERROR: " + err.Error()
	}

	if node.Topics == nil {
		node.Topics = make(map[string]*topicStruct)
	}

	node.Topics[topic]=&topicStruct{Topic: t, Sub: sub}

	// Запускаем readLoop для обработки входящих сообщений
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				return
			}
			if msg.GetFrom() == node.Node.ID() {
				continue
			}
			receiverMutex.RLock()
			if messageReceiver != nil {
				messageReceiver.OnMessageReceived(msg.ReceivedFrom.String(), string(msg.Message.Data))
			}
			receiverMutex.RUnlock()
		}
	}()

	return "успешно подписан на топик: {" + topic + "}"
}


// Получение адресов ноды для отображения в Kotlin
func GetNodeInfo() string {
	if node == nil {
		return ""
	}

	// Берём актуальные адреса у хоста
	peerInfo := peer.AddrInfo{
		ID:    node.Node.ID(),
		Addrs: node.Node.Addrs(),
	}
	// return peerInfo.String()
	addrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		return ""
	}

	out := []string{}

	for _, a := range addrs {
		out = append(out, a.String())
	}

	b, _ := json.Marshal(NodeInfo{
		ID:    node.Node.ID().String(),
		Addrs: out,
	})
	return string(b)
}

