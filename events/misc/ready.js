const { Events, ActivityType } = require('discord.js');
const setting = require('../../config.json');
module.exports = {
    name: Events.ClientReady,
    async execute(client) {
        client.user.setPresence({ activities: [{ name: `${setting.discord.name}`, type: ActivityType.Streaming }], status: 'online' });
        console.log('[成功]', `${client.user.tag}にログインしました。`)
    }
}