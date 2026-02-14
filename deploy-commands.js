const fs = require('node:fs');
const path = require('node:path');
const { REST, Routes } = require('discord.js');
const setting = require('./config.json');


const commands = [];
fs.readdirSync('./commands/').forEach(async dir => {
    const commandsPath = path.join(__dirname, `./commands/${dir}`);
    const commandFiles = fs.readdirSync(commandsPath).filter(file => file.endsWith('.js'));

    for (const file of commandFiles) {
        const command = require(`./commands/${dir}/${file}`);
        commands.push(command.data.toJSON());
    };
});

const rest = new REST({ version: '10' }).setToken(setting.discord.token);

(async () => {
    try {
        console.log("[ロード開始]", `${commands.length} 個のアプリケーション (/) コマンドをロードします。`);
        const data = await rest.put(
            Routes.applicationCommands(setting.discord.clientid),
            { body: commands },
        );

        console.log("[成功]", `${data.length} 個のアプリケーション (/) コマンドをロードしました。`);
    } catch (error) {
        console.error(error);
    }
})();