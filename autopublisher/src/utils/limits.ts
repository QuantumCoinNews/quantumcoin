// Platform karakter/boyut sınırları

export const Limits = {
  x: {
    text: 280,           // X (Twitter) text sınırı
  },
  telegram: {
    caption: 1024,       // Telegram caption sınırı
  },
  youtube: {
    title: 100,
    description: 5000,
    tags: 500,           // toplam karakter
  },
  instagram: {
    caption: 2200,
  },
  tiktok: {
    caption: 150,
  }
};

// Metni kırp
export function enforceLimit(text: string, limit: number): string {
  return text.length > limit ? text.slice(0, limit - 3) + '...' : text;
}
