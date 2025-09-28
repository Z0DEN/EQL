import 'react-native-get-random-values';
export declare class P2PNode {
    #private;
    private node;
    private bootstrapPeers;
    private onMessage?;
    constructor(bootstrapPeers?: string[]);
    /** 🔹 Создать и запустить ноду */
    start(addrToDial?: string | null, onMessage?: (from: string, text: string) => void): Promise<void>;
    /** 🔹 Подписка на топик */
    subscribe(topic: string): void;
    /** 🔹 Публикация сообщения */
    publish(content: string, topic?: string): void;
    /** 🔹 Подключение к другому узлу */
    dial(addr: string): Promise<void>;
    /** 🔹 Завершение работы */
    stop(): Promise<void>;
}
