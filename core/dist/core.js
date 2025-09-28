var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _P2PNode_instances, _P2PNode_setupEvents;
import 'react-native-get-random-values';
import { createLibp2p } from 'libp2p';
import { webSockets } from '@libp2p/websockets';
import { bootstrap } from '@libp2p/bootstrap';
import { noise } from '@chainsafe/libp2p-noise';
import { gossipsub } from '@chainsafe/libp2p-gossipsub';
// import { pubsubPeerDiscovery } from '@libp2p/pubsub-peer-discovery'
// import { circuitRelayTransport } from '@libp2p/circuit-relay-v2'
import { identify } from '@libp2p/identify';
import { multiaddr } from '@multiformats/multiaddr';
import { yamux } from '@chainsafe/libp2p-yamux';
import { EventTarget } from 'event-target-shim';
global.EventTarget = EventTarget;
const TOPIC = 'p2p-social';
export class P2PNode {
    constructor(bootstrapPeers = ['']) {
        _P2PNode_instances.add(this);
        this.bootstrapPeers = bootstrapPeers;
    }
    /** 🔹 Создать и запустить ноду */
    async start(addrToDial = null, onMessage) {
        this.onMessage = onMessage;
        this.node = await createLibp2p({
            transports: [webSockets(),],
            addresses: {
                listen: [
                    '/ip4/0.0.0.0/tcp/0',
                    '/ip4/0.0.0.0/tcp/0/ws',
                    '/p2p-circuit'
                ]
            },
            connectionEncrypters: [noise()],
            streamMuxers: [yamux()],
            connectionGater: { denyDialMultiaddr: async () => false },
            peerDiscovery: [
                bootstrap({ list: this.bootstrapPeers }),
                // pubsubPeerDiscovery({
                //     interval: 10_000,
                //     topics: [TOPIC],
                // }),
            ],
            services: {
                pubsub: gossipsub({ allowPublishToZeroTopicPeers: true }),
                identify: identify()
            }
        });
        await this.node.start();
        console.log(`✅ Node started with id ${this.node.peerId.toString()}`);
        console.log('Listening on:', this.node.getMultiaddrs().map((a) => a.toString()));
        // если нужно сразу подключиться к какому-то адресу
        if (addrToDial)
            await this.dial(addrToDial);
        __classPrivateFieldGet(this, _P2PNode_instances, "m", _P2PNode_setupEvents).call(this);
        this.subscribe(TOPIC); // подписываемся на топик по умолчанию
    }
    /** 🔹 Подписка на топик */
    subscribe(topic) {
        if (!this.node)
            return;
        this.node.services.pubsub.subscribe(topic);
    }
    /** 🔹 Публикация сообщения */
    publish(content, topic = TOPIC) {
        if (!this.node)
            return;
        const data = new TextEncoder().encode(content);
        this.node.services.pubsub.publish(topic, data);
    }
    /** 🔹 Подключение к другому узлу */
    async dial(addr) {
        try {
            const ma = multiaddr(addr);
            await this.node.dial(ma);
            console.log(`🔗 Dialed to ${addr}`);
        }
        catch (e) {
            console.error('Dial error:', e);
        }
    }
    /** 🔹 Завершение работы */
    async stop() {
        if (!this.node)
            return;
        console.log('Stopping node...');
        await this.node.stop();
        console.log('Node stopped');
    }
}
_P2PNode_instances = new WeakSet(), _P2PNode_setupEvents = function _P2PNode_setupEvents() {
    // автоматический коннект к найденным пирам
    this.node.addEventListener('peer:discovery', async (evt) => {
        const maddrs = evt.detail.multiaddrs.map((ma) => ma.encapsulate(`/p2p/${evt.detail.id.toString()}`));
        console.log(`👀 Discovered peer ${evt.detail.id.toString()}`, maddrs.map((m) => m.toString()));
        try {
            await this.node.dial(maddrs);
        }
        catch (err) {
            //console.error('Failed to dial peer:', err)
        }
    });
    // входящие сообщения
    this.node.services.pubsub.addEventListener('message', (evt) => {
        const msg = evt.detail;
        const text = new TextDecoder().decode(msg.data);
        //console.log(`>${msg.from.toString().slice(-5)}: ${text}`)
        this.onMessage?.(msg.from.toString(), text);
    });
};
//# sourceMappingURL=core.js.map