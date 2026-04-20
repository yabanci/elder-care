'use client';

import { useEffect, useRef, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Message } from '@/lib/api';
import { Send } from 'lucide-react';

export default function CareThread({ params }: { params: { otherID: string } }) {
  const { otherID } = params;
  const user = useAuthedUser(['doctor', 'family']);
  const [messages, setMessages] = useState<Message[]>([]);
  const [body, setBody] = useState('');
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!user) return;
    load();
    const id = setInterval(load, 5000);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user, otherID]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length]);

  async function load() {
    const m = await api<Message[]>(`/api/messages/${otherID}`);
    setMessages(m);
  }

  async function send(e: React.FormEvent) {
    e.preventDefault();
    if (!body.trim()) return;
    await api('/api/messages', {
      method: 'POST',
      body: JSON.stringify({ recipient_id: otherID, body }),
    });
    setBody('');
    load();
  }

  if (!user) return null;

  const otherName = messages.find((m) => m.sender_id === otherID)?.sender_name ?? 'Пациент';

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">{otherName}</h1>
      <div className="space-y-2 mb-4">
        {messages.map((m) => {
          const mine = m.sender_id === user.id;
          return (
            <div key={m.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-[85%] rounded-2xl px-4 py-3 ${
                  mine ? 'bg-primary-600 text-white' : 'bg-white border border-ink-300'
                }`}
              >
                <div>{m.body}</div>
                <div className={`text-xs mt-1 ${mine ? 'text-primary-100' : 'text-ink-500'}`}>
                  {new Date(m.created_at).toLocaleTimeString('ru-RU', {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </div>
              </div>
            </div>
          );
        })}
        <div ref={endRef} />
      </div>

      <form onSubmit={send} className="fixed bottom-20 left-0 right-0 bg-white border-t border-ink-300 p-3">
        <div className="max-w-6xl mx-auto flex gap-2">
          <input
            className="field"
            placeholder="Сообщение..."
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          <button type="submit" className="btn-primary !px-4">
            <Send className="w-5 h-5" />
          </button>
        </div>
      </form>
    </Shell>
  );
}
