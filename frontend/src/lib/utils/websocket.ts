export class TerminalWebSocket {
    private ws: WebSocket | null = null;
    private url: string;
    
    constructor(port: number, sessionId: string, token: string) {
        this.url = `ws://127.0.0.1:${port}/ws/${sessionId}?token=${token}`;
    }

    connect(onData: (data: Uint8Array) => void, onClose: () => void, onError: (err: any) => void) {
        this.ws = new WebSocket(this.url);
        this.ws.binaryType = 'arraybuffer';

        this.ws.onopen = () => {
            console.log('Connected to terminal PTY');
        };

        this.ws.onmessage = (event) => {
            if (event.data instanceof ArrayBuffer) {
                onData(new Uint8Array(event.data));
            }
        };

        this.ws.onclose = () => {
            console.log('Terminal PTY disconnected');
            onClose();
        };

        this.ws.onerror = (err) => {
            console.error('Terminal PTY error', err);
            onError(err);
        };
    }

    send(data: string) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(new TextEncoder().encode(data));
        }
    }

    close() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}
