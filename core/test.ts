// test.js
import { P2PNode } from './core.ts'


async function main() {
    const addr = process.argv[2] || null
    // Создаём ноду
    const node = new P2PNode()

    // Запускаем и подписываемся на топик
    await node.start(addr, (from, text) => {
        console.log(`📩 Получено от ${from.slice(-6)}: ${text}`)
    })

    const interval = setInterval(() => {
        const msg = `Hello from interval at ${new Date().toLocaleTimeString()}`
        node.publish(msg)
        console.log('➡️ Отправлено:', msg)
    }, 3000)
}

main().catch(err => console.error(err))
