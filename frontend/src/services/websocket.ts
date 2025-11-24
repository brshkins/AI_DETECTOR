
interface WebSocketMessage {
    type: string;
    payload?: any;
    clientID?: string;
    timestamp?: number;
}

type MessageHandler = (data: any) => void;

class WebSocketService {
    private ws: WebSocket | null = null;
    private url: string;
    private messageHandlers: Map<string, MessageHandler[]> = new Map();
    private reconnectAttempts = 0;
    private maxReconnectAttempts = 10;
    private reconnectTimeout = 3000;
    private shouldReconnect = true;

    constructor(url?: string) {
        // Используем относительный URL для WebSocket, чтобы cookies отправлялись автоматически
        if (url) {
            this.url = url;
        } else {
            // Определяем WebSocket URL на основе текущего протокола и хоста
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            this.url = `${protocol}//${host}/ws`;
        }
    }

    connect(): void {
        if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) {
            console.log('WebSocket already connecting or connected, skipping...');
            return;
        }

        // WebSocket автоматически отправляет cookies для того же домена
        // HttpOnly cookies не видны в document.cookie, но они отправляются автоматически
        this.shouldReconnect = true;
        this.reconnectAttempts = 0; // Сбрасываем счетчик при новом подключении
        console.log('Attempting WebSocket connection to:', this.url);
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log('WebSocket connected, readyState:', this.ws?.readyState);
            this.reconnectAttempts = 0;
            this.trigger('open');
            // Не закрываем соединение - ждем сообщения от сервера
        };

        this.ws.onmessage = (event) => {
            try {
                // WebSocket API автоматически обрабатывает PING/PONG на уровне протокола
                if (typeof event.data === 'string') {
                    const data: WebSocketMessage = JSON.parse(event.data);
                    this.handleMessage(data);
                } else {
                    // Игнорируем бинарные сообщения (PING/PONG обрабатываются автоматически)
                    console.log('Received binary message (likely PING/PONG)');
                }
            } catch (err) {
                console.error('Invalid WebSocket message:', event.data, err);
            }
        };

        this.ws.onclose = (event) => {
            console.log('WebSocket disconnected:', event.code, event.reason, 'clean:', event.wasClean);
            this.ws = null; // Очищаем ссылку на закрытое соединение
            
            // Код 1006 (abnormal closure) может означать, что сервер отклонил соединение (например, 401)
            // Код 1008 (policy violation) может означать проблему с авторизацией
            // Не переподключаемся при этих ошибках
            if (event.code === 1006 || event.code === 1008 || event.code === 1002) {
                console.log('WebSocket connection rejected by server (likely authentication issue). Stopping reconnection.');
                this.shouldReconnect = false;
            }
            
            this.trigger('close');
            if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
                setTimeout(() => {
                    this.reconnectAttempts++;
                    console.log(`Reconnecting... attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                    this.connect();
                }, this.reconnectTimeout);
            } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
                console.log('Max reconnection attempts reached. Stopping reconnection.');
            }
        };

        this.ws.onerror = (err) => {
            console.error('WebSocket error:', err);
            // НЕ закрываем соединение при ошибке - браузер сам обработает
            this.trigger('error', err);
        };
    }

    private handleMessage(message: WebSocketMessage) {
        const handlers = this.messageHandlers.get(message.type) || [];
        handlers.forEach((handler) => handler(message.payload));
    }

    on(type: string, callback: MessageHandler) {
        if (!this.messageHandlers.has(type)) {
            this.messageHandlers.set(type, []);
        }
        this.messageHandlers.get(type)?.push(callback);
    }

    off(type: string, callback: MessageHandler) {
        const handlers = this.messageHandlers.get(type) || [];
        const index = handlers.indexOf(callback);
        if (index > -1) {
            handlers.splice(index, 1);
        }
    }

    send(type: string, payload?: any) {
        if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(
                JSON.stringify({
                    type,
                    payload,
                })
            );
        } else {
            console.warn('WebSocket is not open. Current state:', this.ws?.readyState);
        }
    }

    trigger(type: string, payload?: any) {
        const handlers = this.messageHandlers.get(type) || [];
        handlers.forEach((handler) => handler(payload));
    }

    disconnect() {
        this.shouldReconnect = false;
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    isConnected(): boolean {
        return this.ws?.readyState === WebSocket.OPEN;
    }

    // Метод для переподключения после авторизации
    reconnectAfterAuth(): void {
        if (!this.isConnected()) {
            this.shouldReconnect = true;
            this.reconnectAttempts = 0;
            this.connect();
        }
    }
}

// Singleton instance
export const wsService = new WebSocketService();