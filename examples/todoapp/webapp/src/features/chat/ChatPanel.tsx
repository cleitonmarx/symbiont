import { useEffect, useRef, useState } from 'react';
import { marked } from 'marked';
import { useChat } from '../../hooks/useChat';

marked.setOptions({
  breaks: true,
  gfm: true,
});

interface ChatPanelProps {
  onChatDone?: () => void;
  mode?: 'panel' | 'sheet';
  onClose?: () => void;
}

export const ChatPanel = ({ onChatDone, mode = 'panel', onClose }: ChatPanelProps) => {
  const {
    messages,
    models,
    selectedModel,
    toolStatus,
    toolStatusCount,
    loading,
    loadingModels,
    error,
    loadMessages,
    loadModels,
    setSelectedModel,
    sendMessage,
    clearChat,
    stopStream,
  } = useChat(onChatDone);
  const [input, setInput] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const isInitialLoad = useRef(true);

  useEffect(() => {
    loadMessages();
    loadModels();
  }, [loadMessages, loadModels]);

  useEffect(() => {
    if (messages.length > 0 && messagesContainerRef.current) {
      if (isInitialLoad.current) {
        messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
        isInitialLoad.current = false;
      } else {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    }
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim()) {
      return;
    }
    await sendMessage(input);
    setInput('');
  };

  return (
    <section className={`ui-chat ${mode}`}>
      <header className="ui-chat-header">
        <div>
          <h2>AI Assistant</h2>
          <p>Ask about todos, priorities, and updates</p>
        </div>
        <div className="ui-chat-header-actions">
          <button type="button" className="ui-icon-btn" onClick={clearChat} title="Clear chat history" aria-label="Clear chat history">
            🗑
          </button>
          {onClose ? (
            <button type="button" className="ui-icon-btn" onClick={onClose} title="Close chat" aria-label="Close chat">
              ✕
            </button>
          ) : null}
        </div>
      </header>

      <div className="ui-chat-messages" ref={messagesContainerRef}>
        {error ? <div className="ui-chat-error">{error}</div> : null}

        {messages.map((message) => (
          <article key={message.id} className={`ui-chat-message ${message.role}`}>
            <div className="ui-chat-message-content">
              {message.role === 'assistant' ? (
                <div dangerouslySetInnerHTML={{ __html: marked(message.content) }} />
              ) : (
                message.content
              )}
            </div>
            <time>{new Date(message.created_at).toLocaleTimeString()}</time>
          </article>
        ))}

        {loading ? (
          <article className="ui-chat-message assistant">
            <div className="ui-chat-typing">
              <span />
              <span />
              <span />
            </div>
          </article>
        ) : null}

        <div ref={messagesEndRef} />
      </div>

      {toolStatus ? (
        <div className="ui-chat-tool-status">
          <span>{toolStatus}</span>
          <span className="ui-chat-tool-count">x{toolStatusCount}</span>
        </div>
      ) : null}

      <footer className="ui-chat-composer">
        <div className="ui-chat-input-shell">
          <textarea
            className="ui-chat-input"
            value={input}
            disabled={loading}
            placeholder="Ask for follow-up changes"
            onChange={(event) => setInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                handleSend();
              }
            }}
          />

          <div className="ui-chat-input-controls">
            <div className="ui-chat-input-meta">
              <select
                className="ui-chat-model-select"
                aria-label="AI model"
                value={selectedModel}
                disabled={loading || loadingModels || models.length === 0}
                onChange={(event) => setSelectedModel(event.target.value)}
              >
                {models.length === 0 ? (
                  <option value="">{loadingModels ? 'Loading models...' : 'Default model'}</option>
                ) : (
                  models.map((model) => (
                    <option key={model} value={model}>
                      {model}
                    </option>
                  ))
                )}
              </select>
            </div>

            {loading ? (
              <button type="button" className="ui-chat-send-btn stop" onClick={stopStream} aria-label="Stop stream">
                ■
              </button>
            ) : (
              <button
                type="button"
                className="ui-chat-send-btn"
                onClick={handleSend}
                disabled={!input.trim()}
                aria-label="Send message"
              >
                ↑
              </button>
            )}
          </div>
        </div>
      </footer>
    </section>
  );
};
