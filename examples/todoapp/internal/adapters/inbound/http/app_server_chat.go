package http

import (
	"net/http"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
)

func (api TodoAppServer) ClearChatMessages(w http.ResponseWriter, r *http.Request) {
	err := api.DeleteConversationUseCase.Execute(r.Context())
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api TodoAppServer) ListChatMessages(w http.ResponseWriter, r *http.Request, params openapi.ListChatMessagesParams) {
	messages, hasMore, err := api.ListChatMessagesUseCase.Query(r.Context(), params.Page, params.Pagesize)
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	resp := openapi.ChatHistoryResp{
		ConversationId: domain.GlobalConversationID,
		Messages:       []openapi.ChatMessage{},
		Page:           params.Page,
	}
	if hasMore {
		nextPage := params.Page + 1
		resp.NextPage = &nextPage
	}
	if params.Page > 1 {
		prevPage := params.Page - 1
		resp.PreviousPage = &prevPage
	}

	for _, msg := range messages {
		resp.Messages = append(resp.Messages, openapi.ChatMessage{
			Id:        msg.ID,
			Role:      openapi.ChatMessageRole(msg.ChatRole),
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, resp)

}

func (api TodoAppServer) StreamChat(w http.ResponseWriter, r *http.Request) {

}
