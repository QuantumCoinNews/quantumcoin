package main

type Publishers struct {
	Tele *TelegramPublisher
	// İleride: X, YouTube, Instagram vb.
}

func NewPublishers(a *Auth) *Publishers {
	return &Publishers{
		Tele: NewTelegramPublisher(a),
	}
}

func (p *Publishers) PublishAll(item ContentItem) error {
	// Şimdilik Telegram gerçek gönderim yapar
	if err := p.Tele.Publish(item); err != nil {
		return err
	}
	// TODO: diğer platformlar entegre olduğunda sırayla çağır
	// _ = PublishX(item)
	// _ = PublishYouTube(item)
	// _ = PublishInstagram(item)
	return nil
}
