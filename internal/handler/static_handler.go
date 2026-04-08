package handler

import (
	"e-commerce/views/public"
	"net/http"
)

type StaticHandler struct{}

func NewStaticHandler() *StaticHandler {
	return &StaticHandler{}
}

func getFrom(r *http.Request) string {
	from := r.URL.Query().Get("from")
	if from != "" {
		return from
	}

	referer := r.Header.Get("Referer")
	if referer != "" {
		return referer
	}

	return "/"
}

func (h *StaticHandler) TentangKami(w http.ResponseWriter, r *http.Request) {
	public.TentangKamiPage(getFrom(r)).Render(r.Context(), w)
}

func (h *StaticHandler) CaraBelanja(w http.ResponseWriter, r *http.Request) {
	public.CaraBelanjaPage(getFrom(r)).Render(r.Context(), w)
}

func (h *StaticHandler) MetodePembayaran(w http.ResponseWriter, r *http.Request) {
	public.MetodePembayaranPage(getFrom(r)).Render(r.Context(), w)
}

func (h *StaticHandler) Pengiriman(w http.ResponseWriter, r *http.Request) {
	public.PengirimanPage(getFrom(r)).Render(r.Context(), w)
}

func (h *StaticHandler) SyaratKetentuan(w http.ResponseWriter, r *http.Request) {
	public.SyaratKetentuanPage(getFrom(r)).Render(r.Context(), w)
}
