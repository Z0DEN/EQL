import { createLibp2p } from 'libp2p'
import { webSockets } from '@libp2p/websockets'
import { bootstrap } from '@libp2p/bootstrap'
import { noise } from '@chainsafe/libp2p-noise'
import { pubsubPeerDiscovery } from '@libp2p/pubsub-peer-discovery'
import { gossipsub } from '@chainsafe/libp2p-gossipsub'
import { circuitRelayTransport } from '@libp2p/circuit-relay-v2'
import { identify } from '@libp2p/identify'
import { multiaddr } from '@multiformats/multiaddr'
import { yamux } from '@chainsafe/libp2p-yamux'
import readline from 'readline'
import { tcp } from '@libp2p/tcp'

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
})

const TOPIC = 'p2p-social'
const bootstrapPeers = [
     '',
]

export async function startNode(addrToDial=0) {
    const node = await createLibp2p({
        transports: [
            webSockets(),
            circuitRelayTransport(),
            tcp(),
        ],
        addresses: { listen: [
           '/ip4/0.0.0.0/tcp/0',
           '/ip4/0.0.0.0/tcp/0/ws',
           '/p2p-circuit',
        ] },
        connectionEncrypters: [noise()],
        streamMuxers: [yamux()],
        connectionGater: {
        // Allow private addresses for local testing
        denyDialMultiaddr: async () => false,
        },
        peerDiscovery: [
        bootstrap({
          list: [''],
        }),
        // pubsubPeerDiscovery({
        //   interval: 10_000,
        //   topics: [TOPIC],
        // }),
        ],
        services: {
            pubsub: gossipsub({allowPublishToZeroTopicPeers: true,}),
            identify: identify(),
        }

    })

    await node.start()
    console.log('Node started with id', node.peerId.toString())
    console.log(`Listening on: ${node.getMultiaddrs().map(a => a.toString())}`)
    if (addrToDial) {
        const ma = multiaddr(addrToDial)
        try {
            await node.dial(ma)
        } catch (e) {
            console.log(e)
        }
    }

    node.addEventListener('peer:discovery', async (evt) => {
        // Encapsulate the multiaddrs with the peer ID to ensure correct dialing
        // Should be fixed when https://github.com/libp2p/js-libp2p/issues/3239 is resolved.
        const maddrs = evt.detail.multiaddrs.map((ma) => ma.encapsulate(`/p2p/${evt.detail.id.toString()}`))
        console.log(
            `Discovered new peer (${evt.detail.id.toString()}). Dialling:`, maddrs.map((ma) => ma.toString()),
        )
        try {
            await node.dial(maddrs) // dial the new peer
        } catch (err) {
            console.error(`Failed to dial peer (${evt.detail.id.toString()}):`, err)
        }
    })

    // Подписка на топик
    node.services.pubsub.subscribe(TOPIC)
    node.services.pubsub.addEventListener('message', (evt) => {
        const msg = evt.detail
        console.log(`>${msg.from.toString().slice(-5)}: ${new TextDecoder().decode(msg.data)}`)
    })


    rl.on('line', (input) => {
        if (input.trim() === '') return
            publishMessage(input.trim())
    })


    process.on('SIGINT', async () => {
        console.log('Stopping node...')
        await node.stop()
        console.log('Node stopped')
        process.exit(0)
    })
}

const publishMessage = (content) => {
    const data = new TextEncoder().encode(content)
    node.services.pubsub.publish(TOPIC, data)
}

const args = process.argv.slice(2)
const addrToDial = args[0]
startNode(addrToDial)
