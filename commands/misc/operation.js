const { SlashCommandBuilder, EmbedBuilder, PermissionsBitField } = require('discord.js');
const model = require('../../libs/mongo');
const Operation = model.operation;
const message = new EmbedBuilder()
    .setColor(0x0099FF)
    .setTitle('お知らせ')
module.exports = {
    data: new SlashCommandBuilder()
        .setName('operation')
        .setDMPermission(false)
        .addStringOption(option => option.setName('select').setDescription('操作を入力してください').addChoices({ name: '開始', value: 'start' }, { name: '停止', value: 'stop' }).setRequired(true))
        .setDescription('開始/停止します'),
    async execute(interaction) {
        if (!interaction.member.permissions.has(PermissionsBitField.Flags.Administrator)) return;
        const data = await Operation.find({ serverid: interaction.guild.id });
        if (interaction.options.getString('select') === 'start') {
            if (data[0]?.serverid) return await interaction.reply('登録済み');
            const Server = new Operation({
                serverid: interaction.guild.id,
                channelid: interaction.channel.id
            });
            await Server.save();
            await interaction.reply({ embeds: [message.setDescription('保存しました。')] })
        } else if (interaction.options.getString('select') === 'stop') {
            if (!data[0]?.serverid) return await interaction.reply('未登録');
            await Operation.findByIdAndDelete(data[0].id);
            await interaction.reply({ embeds: [message.setDescription('削除しました。')] })
        } else {
            await interaction.reply('不明なエラー');
        };
    },
};