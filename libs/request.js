const Parser = require('rss-parser');
const parser = new Parser();
const model = require('./mongo');
const Operation = model.operation;
const Tmp = model.tmp;
const setting = require('../config.json');

// クエリパラメータを削除する関数
const cleanLink = url => {
  try {
    const urlObj = new URL(url);

    // `utm_` で始まるクエリパラメータを削除
    const params = new URLSearchParams(urlObj.search);
    for (const [key] of params.entries()) {
      if (key.startsWith('utm_')) {
        params.delete(key);
      }
    }
    urlObj.search = params.toString(); // 残ったクエリパラメータを再設定

    return urlObj.toString();
  } catch (error) {
    console.error(`Invalid URL: ${url}`);
    return url; // 無効なURLはそのまま返す
  }
};

const send = async client => {
  try {
    for (const link of setting.RSS.links) {
      console.log(`Fetching RSS feed from: ${link}`);
      const feed = await parser.parseURL(link);
      for (const item of feed.items) {
        if (!item?.link) continue;
        const cleanedLink = cleanLink(item.link); // リンクをクリーンアップ
        console.log(`Processing item: ${cleanedLink}`);
        const links = await Tmp.find({ link: cleanedLink });
        if (links.length > 0) continue;
        const tmp = new Tmp({ link: cleanedLink });
        await tmp.save();
        const datas = await Operation.find({});
        for (const data of datas) {
          try {
            const guild = await client.guilds.fetch(data.serverid);
            if (!guild) continue;
            const channel = await guild.channels.fetch(data.channelid);
            if (!channel) continue;
            await channel.send(`${item.title}\n${cleanedLink}`);
          } catch (guildError) {
            console.error(`Failed to send message to guild: ${data.serverid}`, guildError);
          }
        }
      }
    }
  } catch (error) {
    console.error('Error in send function:', error);
  }
};

module.exports = {
  async search(client) {
    try {
      await send(client);
      setInterval(async () => await send(client), 5 * 60 * 1000);
    } catch (error) {
      console.error('Error in search function:', error);
    }
  }
};
