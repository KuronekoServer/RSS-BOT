const mongoose = require('mongoose');
const Schema = mongoose.Schema;
const setting = require('../config.json');
const operation = new Schema({
  serverid: String,
  channelid: String
});
const tmp = new Schema({
  link: String
});
mongoose.connect(setting.mongo.link)
  .then(() => console.log('Connected!'));
exports.operation = mongoose.model('operation', operation);
exports.tmp = mongoose.model('tmp', tmp);