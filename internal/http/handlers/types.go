package handlers

// ErrorResponse representa uma resposta de erro padrão da API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse representa uma resposta de sucesso padrão da API
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// PaginatedResponse representa uma resposta paginada
type PaginatedResponse struct {
	Data       any `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// HealthResponse representa a resposta do health check
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
}

// SendImageHTTPRequest representa uma requisição HTTP para envio de imagem
type SendImageHTTPRequest struct {
	ToJID     string `json:"to_jid" form:"to_jid" binding:"required"`
	Caption   string `json:"caption,omitempty" form:"caption"`
	ReplyToID string `json:"reply_to_id,omitempty" form:"reply_to_id"`
	// Para upload de arquivo
	ImageFile interface{} `json:"-" form:"image_file"`
	// Para base64
	ImageBase64 string `json:"image_base64,omitempty" form:"image_base64"`
	// Para URL pública
	ImageURL string `json:"image_url,omitempty" form:"image_url"`
	// Metadados opcionais
	FileName string `json:"file_name,omitempty" form:"file_name"`
	MimeType string `json:"mime_type,omitempty" form:"mime_type"`
}

// SendVideoHTTPRequest representa uma requisição HTTP para envio de vídeo
type SendVideoHTTPRequest struct {
	ToJID     string `json:"to_jid" form:"to_jid" binding:"required"`
	Caption   string `json:"caption,omitempty" form:"caption"`
	ReplyToID string `json:"reply_to_id,omitempty" form:"reply_to_id"`
	// Para upload de arquivo
	VideoFile interface{} `json:"-" form:"video_file"`
	// Para base64
	VideoBase64 string `json:"video_base64,omitempty" form:"video_base64"`
	// Para URL pública
	VideoURL string `json:"video_url,omitempty" form:"video_url"`
	// Metadados opcionais
	FileName string `json:"file_name,omitempty" form:"file_name"`
	MimeType string `json:"mime_type,omitempty" form:"mime_type"`
}

// SendAudioHTTPRequest representa uma requisição HTTP para envio de áudio
type SendAudioHTTPRequest struct {
	ToJID     string `json:"to_jid" form:"to_jid" binding:"required"`
	ReplyToID string `json:"reply_to_id,omitempty" form:"reply_to_id"`
	IsVoice   bool   `json:"is_voice,omitempty" form:"is_voice"`
	// Para upload de arquivo
	AudioFile interface{} `json:"-" form:"audio_file"`
	// Para base64
	AudioBase64 string `json:"audio_base64,omitempty" form:"audio_base64"`
	// Para URL pública
	AudioURL string `json:"audio_url,omitempty" form:"audio_url"`
	// Metadados opcionais
	FileName string `json:"file_name,omitempty" form:"file_name"`
	MimeType string `json:"mime_type,omitempty" form:"mime_type"`
}

// SendDocumentHTTPRequest representa uma requisição HTTP para envio de documento
type SendDocumentHTTPRequest struct {
	ToJID     string `json:"to_jid" form:"to_jid" binding:"required"`
	ReplyToID string `json:"reply_to_id,omitempty" form:"reply_to_id"`
	// Para upload de arquivo
	DocumentFile interface{} `json:"-" form:"document_file"`
	// Para base64
	DocumentBase64 string `json:"document_base64,omitempty" form:"document_base64"`
	// Para URL pública
	DocumentURL string `json:"document_url,omitempty" form:"document_url"`
	// Metadados obrigatórios para documentos
	FileName string `json:"file_name" form:"file_name" binding:"required"`
	MimeType string `json:"mime_type,omitempty" form:"mime_type"`
}

// SendStickerHTTPRequest representa uma requisição HTTP para envio de sticker
type SendStickerHTTPRequest struct {
	ToJID     string `json:"to_jid" form:"to_jid" binding:"required"`
	ReplyToID string `json:"reply_to_id,omitempty" form:"reply_to_id"`
	// Para upload de arquivo
	StickerFile interface{} `json:"-" form:"sticker_file"`
	// Para base64
	StickerBase64 string `json:"sticker_base64,omitempty" form:"sticker_base64"`
	// Para URL pública
	StickerURL string `json:"sticker_url,omitempty" form:"sticker_url"`
	// Metadados opcionais
	FileName string `json:"file_name,omitempty" form:"file_name"`
	MimeType string `json:"mime_type,omitempty" form:"mime_type"`
}
