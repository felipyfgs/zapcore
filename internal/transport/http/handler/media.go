package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	entity "wamex/internal/domain/entity"
	domainRepo "wamex/internal/domain/repository"
	"wamex/internal/infra/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// MediaHandler gerencia operações de mídia
type MediaHandler struct {
	mediaRepo   domainRepo.MediaRepository
	minioClient *storage.MinIOClient
	mediaBucket string
}

// NewMediaHandler cria uma nova instância do handler de mídia
func NewMediaHandler(mediaRepo domainRepo.MediaRepository, minioClient *storage.MinIOClient) *MediaHandler {
	return &MediaHandler{
		mediaRepo:   mediaRepo,
		minioClient: minioClient,
		mediaBucket: "wamex-media", // TODO: pegar do config
	}
}

// UploadMedia implementa POST /media/upload
func (h *MediaHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Iniciando upload de mídia")

	// Limita o tamanho da requisição (100MB + overhead)
	r.Body = http.MaxBytesReader(w, r.Body, entity.MaxDocumentSize+1024*1024)

	// Parse do multipart form
	if err := r.ParseMultipartForm(entity.MaxDocumentSize); err != nil {
		log.Error().Err(err).Msg("Erro ao fazer parse do multipart form")
		h.writeErrorResponse(w, http.StatusBadRequest, "FORM_TOO_LARGE", "Arquivo muito grande", "file")
		return
	}

	// Obtém o arquivo do form
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Error().Err(err).Msg("Arquivo não encontrado no form")
		h.writeErrorResponse(w, http.StatusBadRequest, "MISSING_FILE", "Arquivo é obrigatório", "file")
		return
	}
	defer file.Close()

	// Validação básica do arquivo
	if header.Size > 50*1024*1024 { // 50MB limit
		log.Error().Str("filename", header.Filename).Int64("size", header.Size).Msg("Arquivo muito grande")
		h.writeErrorResponse(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File size exceeds 50MB limit", "file")
		return
	}

	// Criar estrutura de mídia básica
	mediaFile := &entity.MediaFile{
		ID:       uuid.New().String(),
		Filename: filepath.Base(header.Filename),
		Size:     header.Size,
		MimeType: header.Header.Get("Content-Type"),
	}

	// Sanitiza o nome do arquivo (implementação básica)
	if customFilename := r.FormValue("filename"); customFilename != "" {
		mediaFile.Filename = filepath.Base(customFilename)
	}

	// Captura informações da sessão se fornecidas (opcional)
	if sessionID := r.FormValue("sessionId"); sessionID != "" {
		mediaFile.SessionID = sessionID
	}
	if sessionName := r.FormValue("sessionName"); sessionName != "" {
		mediaFile.SessionName = sessionName
	}

	// Lê os dados do arquivo
	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao ler dados do arquivo")
		h.writeErrorResponse(w, http.StatusInternalServerError, "READ_ERROR", "Erro ao processar arquivo", "file")
		return
	}

	// Gera caminho no MinIO (estrutura hierárquica por data)
	now := time.Now()
	objectPath := fmt.Sprintf("%d/%02d/%02d/%s_%s",
		now.Year(), now.Month(), now.Day(),
		mediaFile.ID, mediaFile.Filename)

	// Faz upload para o MinIO
	ctx := context.Background()
	minioURL, err := h.minioClient.UploadFile(ctx, h.mediaBucket, objectPath, fileData, mediaFile.MimeType)
	if err != nil {
		log.Error().Err(err).Str("object_path", objectPath).Msg("Erro ao fazer upload para MinIO")
		h.writeErrorResponse(w, http.StatusInternalServerError, "UPLOAD_ERROR", "Erro ao salvar arquivo", "")
		return
	}

	// Define o caminho do arquivo no MinIO
	mediaFile.FilePath = objectPath

	// Salva metadados no banco de dados
	if err := h.mediaRepo.Create(ctx, mediaFile); err != nil {
		log.Error().Err(err).Str("media_id", mediaFile.ID).Msg("Erro ao salvar metadados no banco")

		// Tenta remover arquivo do MinIO em caso de erro
		h.minioClient.DeleteFile(ctx, h.mediaBucket, objectPath)

		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Erro ao salvar metadados", "")
		return
	}

	log.Info().
		Str("media_id", mediaFile.ID).
		Str("filename", mediaFile.Filename).
		Str("mime_type", mediaFile.MimeType).
		Int64("size", mediaFile.Size).
		Str("minio_url", minioURL).
		Msg("Upload de mídia realizado com sucesso")

	// Resposta de sucesso
	response := entity.MediaUploadResponse{
		Success: true,
		Message: "Arquivo enviado com sucesso",
		Data:    *mediaFile,
	}

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// DownloadMedia implementa GET /media/{mediaID}/download
func (h *MediaHandler) DownloadMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaID")

	log.Info().Str("media_id", mediaID).Msg("Iniciando download de mídia")

	if mediaID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "MISSING_MEDIA_ID", "ID da mídia é obrigatório", "mediaID")
		return
	}

	// Busca metadados no banco
	ctx := context.Background()
	mediaFile, err := h.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		log.Error().Err(err).Str("media_id", mediaID).Msg("Mídia não encontrada")
		h.writeErrorResponse(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "Mídia não encontrada", "mediaID")
		return
	}

	// Verifica se não expirou
	if time.Now().After(mediaFile.ExpiresAt) {
		log.Warn().Str("media_id", mediaID).Time("expires_at", mediaFile.ExpiresAt).Msg("Mídia expirada")
		h.writeErrorResponse(w, http.StatusGone, "MEDIA_EXPIRED", "Mídia expirada", "mediaID")
		return
	}

	// Baixa arquivo do MinIO
	fileData, err := h.minioClient.DownloadFile(ctx, h.mediaBucket, mediaFile.FilePath)
	if err != nil {
		log.Error().Err(err).Str("media_id", mediaID).Str("file_path", mediaFile.FilePath).Msg("Erro ao baixar arquivo do MinIO")
		h.writeErrorResponse(w, http.StatusInternalServerError, "DOWNLOAD_ERROR", "Erro ao baixar arquivo", "")
		return
	}

	// Define headers apropriados
	w.Header().Set("Content-Type", mediaFile.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(mediaFile.Size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", mediaFile.Filename))
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache por 1 hora

	// Serve o arquivo
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)

	log.Info().
		Str("media_id", mediaID).
		Str("filename", mediaFile.Filename).
		Int64("size", mediaFile.Size).
		Msg("Download de mídia realizado com sucesso")
}

// DeleteMedia implementa DELETE /media/{mediaID}
func (h *MediaHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaID")

	log.Info().Str("media_id", mediaID).Msg("Iniciando deleção de mídia")

	if mediaID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "MISSING_MEDIA_ID", "ID da mídia é obrigatório", "mediaID")
		return
	}

	// Busca metadados no banco
	ctx := context.Background()
	mediaFile, err := h.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		log.Error().Err(err).Str("media_id", mediaID).Msg("Mídia não encontrada para deleção")
		h.writeErrorResponse(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "Mídia não encontrada", "mediaID")
		return
	}

	// Remove arquivo do MinIO
	if err := h.minioClient.DeleteFile(ctx, h.mediaBucket, mediaFile.FilePath); err != nil {
		log.Error().Err(err).Str("media_id", mediaID).Str("file_path", mediaFile.FilePath).Msg("Erro ao remover arquivo do MinIO")
		// Continua mesmo com erro no MinIO para limpar banco
	}

	// Remove metadados do banco
	if err := h.mediaRepo.Delete(ctx, mediaID); err != nil {
		log.Error().Err(err).Str("media_id", mediaID).Msg("Erro ao remover metadados do banco")
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Erro ao remover mídia", "")
		return
	}

	log.Info().
		Str("media_id", mediaID).
		Str("filename", mediaFile.Filename).
		Msg("Mídia removida com sucesso")

	// Resposta de sucesso
	response := map[string]interface{}{
		"success": true,
		"message": "Mídia removida com sucesso",
		"data": map[string]string{
			"id":       mediaID,
			"filename": mediaFile.Filename,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ListMedia implementa GET /media/list
func (h *MediaHandler) ListMedia(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Listando mídias")

	// Parâmetros de paginação e filtros
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	messageType := r.URL.Query().Get("type")
	sessionID := r.URL.Query().Get("sessionId")
	sessionName := r.URL.Query().Get("sessionName")

	// Valores padrão
	limit := 20
	offset := 0

	// Parse dos parâmetros
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Busca mídias no banco
	ctx := context.Background()
	mediaFiles, total, err := h.mediaRepo.List(ctx, limit, offset, messageType, sessionID, sessionName)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao listar mídias")
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Erro ao buscar mídias", "")
		return
	}

	// Calcula informações de paginação
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	log.Info().
		Int("total", total).
		Int("page", page).
		Int("limit", limit).
		Str("message_type", messageType).
		Str("session_id", sessionID).
		Str("session_name", sessionName).
		Msg("Mídias listadas com sucesso")

	// Resposta de sucesso com paginação
	responseWithPagination := map[string]interface{}{
		"success": true,
		"message": "Mídias listadas com sucesso",
		"data":    mediaFiles,
		"pagination": map[string]interface{}{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
			"hasNext":    page < totalPages,
			"hasPrev":    page > 1,
		},
		"filters": map[string]interface{}{
			"messageType": messageType,
			"sessionId":   sessionID,
			"sessionName": sessionName,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, responseWithPagination)
}

// writeErrorResponse escreve uma resposta de erro padronizada
func (h *MediaHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"field":   field,
		},
		"timestamp": time.Now(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// writeJSONResponse escreve uma resposta JSON
func (h *MediaHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
