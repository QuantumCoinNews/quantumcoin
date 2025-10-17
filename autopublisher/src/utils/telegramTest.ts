import axios from "axios";
import dotenv from "dotenv";

dotenv.config();

async function sendTest() {
  const token = process.env.TELEGRAM_BOT_TOKEN;
  const chatId = process.env.TELEGRAM_CHAT_ID;

  if (!token || !chatId) {
    console.error("❌ TELEGRAM_BOT_TOKEN veya TELEGRAM_CHAT_ID bulunamadı (.env kontrol et).");
    process.exit(1);
  }

  const url = `https://api.telegram.org/bot${token}/sendMessage`;

  try {
    const res = await axios.post(url, {
      chat_id: chatId,
      text: "✅ QuantumCoin autopublisher test mesajı başarıyla çalıştı!"
    });
    console.log("📩 Telegram mesajı gönderildi:", res.data);
  } catch (err) {
    console.error("❌ Mesaj gönderilemedi:", err);
  }
}

sendTest();
