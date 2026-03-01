import { useState, useRef, useEffect } from 'react';

// Icons
const SendIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="22" y1="2" x2="11" y2="13"></line>
    <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
  </svg>
);

const BotIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="3" y="11" width="18" height="10" rx="2"></rect>
    <circle cx="12" cy="5" r="2"></circle>
    <path d="M12 7v4"></path>
    <line x1="8" y1="16" x2="8" y2="16"></line>
    <line x1="16" y1="16" x2="16" y2="16"></line>
  </svg>
);

const UserIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
    <circle cx="12" cy="7" r="4"></circle>
  </svg>
);

interface Message {
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
}

function App() {
  const [messages, setMessages] = useState<Message[]>([
    { role: 'assistant', content: 'Hello! I am your Neuron CLI agent. How can I help you today?' }
  ]);
  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isTyping) return;

    const userMsg: Message = { role: 'user', content: input.trim() };
    const newMessages = [...messages, userMsg];
    
    setMessages(newMessages);
    setInput('');
    setIsTyping(true);

    try {
      // In development, the Vite proxy or absolute URL needs to be used
      // For now, assume it's running on the same host, or fallback to dev port
      const apiUrl = window.location.port === '5173' || window.location.port === '5174' 
        ? 'http://127.0.0.1:3133/api/chat' 
        : '/api/chat';

      const response = await fetch(apiUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ messages: newMessages }),
      });

      if (!response.ok) {
        throw new Error(`API error: ${response.status}`);
      }

      setMessages([...newMessages, { role: 'assistant', content: '' }]);

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (reader) {
        let assistantContent = '';
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          const lines = chunk.split('\n');
          
          for (const line of lines) {
            if (line.startsWith('data: ')) {
              try {
                const data = JSON.parse(line.slice(6));
                
                if (data.error) {
                  assistantContent += `\n\n**Error:** ${data.error}`;
                } else if (data.content) {
                  assistantContent += data.content;
                }

                setMessages(prev => {
                  const updated = [...prev];
                  const lastIdx = updated.length - 1;
                  if (updated[lastIdx].role === 'assistant') {
                    updated[lastIdx] = { ...updated[lastIdx], content: assistantContent };
                  }
                  return updated;
                });
              } catch (e) {
                console.error("Failed to parse SSE JSON", e, line);
              }
            }
          }
        }
      }
    } catch (error) {
      console.error('Chat error:', error);
      setMessages(prev => [...prev, { 
        role: 'system', 
        content: `Error connecting to Neuron backend: ${error instanceof Error ? error.message : String(error)}` 
      }]);
    } finally {
      setIsTyping(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <div className="flex flex-col h-screen w-full text-zinc-100 bg-slate-900 overflow-hidden font-sans">
      {/* Header */}
      <header className="flex-none px-6 py-4 flex items-center justify-between border-b border-white/5 bg-slate-900/50 backdrop-blur-md z-10">
        <div className="flex items-center gap-3">
          <div className="bg-indigo-500/20 p-2 rounded-xl text-indigo-400 border border-indigo-500/20 shadow-[0_0_15px_rgba(99,102,241,0.2)]">
            <BotIcon />
          </div>
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-white">Neuron <span className="text-zinc-500 font-light">CLI</span></h1>
            <p className="text-xs text-indigo-400 flex items-center gap-1.5 mt-0.5">
              <span className="w-1.5 h-1.5 rounded-full bg-indigo-400 animate-pulse-subtle"></span>
              Agent Active
            </p>
          </div>
        </div>
      </header>

      {/* Chat Area */}
      <main className="flex-1 overflow-y-auto px-4 py-6 md:px-8 flex flex-col items-center">
        <div className="w-full max-w-4xl flex flex-col gap-6">
          {messages.map((msg, idx) => (
            <div 
              key={idx} 
              className={`flex gap-4 p-5 rounded-2xl animate-fade-in ${
                msg.role === 'user' 
                  ? 'bg-indigo-500/10 border border-indigo-500/20 self-end max-w-[85%]' 
                  : msg.role === 'system'
                  ? 'bg-red-500/10 border border-red-500/20 mx-auto text-sm w-full max-w-lg'
                  : 'bg-white/5 border border-white/5 self-start w-full'
              }`}
            >
              {msg.role !== 'system' && (
                <div className={`flex-none w-8 h-8 rounded-full flex items-center justify-center mt-0.5 ${
                  msg.role === 'user' ? 'bg-indigo-500 text-white' : 'bg-slate-700 text-zinc-300'
                }`}>
                  {msg.role === 'user' ? <UserIcon /> : <BotIcon />}
                </div>
              )}
              
              <div className="flex-1 space-y-2 overflow-hidden">
                <div className="text-sm font-medium text-zinc-400 mb-1">
                  {msg.role === 'user' ? 'You' : msg.role === 'system' ? 'System' : 'Neuron'}
                </div>
                <div className="whitespace-pre-wrap leading-relaxed">
                  {msg.content || (msg.role === 'assistant' && isTyping ? <span className="animate-pulse">...</span> : '')}
                </div>
              </div>
            </div>
          ))}
          {isTyping && messages[messages.length - 1]?.role === 'user' && (
            <div className="flex gap-4 p-5 rounded-2xl bg-white/5 border border-white/5 self-start w-full animate-fade-in">
              <div className="flex-none w-8 h-8 rounded-full bg-slate-700 text-zinc-300 flex items-center justify-center mt-0.5">
                <BotIcon />
              </div>
              <div className="flex items-center gap-1.5 text-zinc-500">
                <span className="w-2 h-2 rounded-full bg-indigo-400 animate-bounce" style={{ animationDelay: '0ms' }}></span>
                <span className="w-2 h-2 rounded-full bg-indigo-400 animate-bounce" style={{ animationDelay: '150ms' }}></span>
                <span className="w-2 h-2 rounded-full bg-indigo-400 animate-bounce" style={{ animationDelay: '300ms' }}></span>
              </div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>
      </main>

      {/* Input Area */}
      <footer className="flex-none p-4 md:p-6 bg-gradient-to-t from-slate-900 via-slate-900/95 to-transparent flex justify-center">
        <div className="w-full max-w-4xl relative">
          <form 
            onSubmit={handleSubmit}
            className="relative flex items-end bg-slate-800/80 backdrop-blur-xl border border-white/10 rounded-2xl p-2 transition-all focus-within:border-indigo-500/50 shadow-lg shadow-black/20"
          >
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Message Neuron... (Shift+Enter for new line)"
              className="w-full max-h-48 min-h-[44px] bg-transparent border-none text-zinc-100 placeholder-zinc-500 focus:ring-0 resize-none py-3 px-4 outline-none"
              rows={Math.min(5, input.split('\n').length || 1)}
              disabled={isTyping}
            />
            <button
              type="submit"
              disabled={!input.trim() || isTyping}
              className={`flex-none mb-1 mr-1 p-3 rounded-xl flex items-center justify-center transition-all ${
                input.trim() && !isTyping
                  ? 'bg-indigo-500 text-white shadow-[0_0_15px_rgba(99,102,241,0.4)] hover:bg-indigo-400 hover:scale-105' 
                  : 'bg-white/5 text-zinc-500 cursor-not-allowed'
              }`}
            >
              <SendIcon />
            </button>
          </form>
          <div className="text-center mt-3 text-xs text-zinc-500">
            Neuron uses local LLMs via Ollama. Data remains on your device.
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
