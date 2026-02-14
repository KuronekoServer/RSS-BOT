# ベースイメージを指定
FROM node:lts

# コンテナ内の作業ディレクトリを設定
WORKDIR /app
# ローカルのpackage.jsonとpackage-lock.jsonをコンテナ内の作業ディレクトリにコピー
COPY package*.json ./

# npmパッケージのインストール
RUN npm install

# ローカルのソースコードをコンテナ内の作業ディレクトリにコピー
COPY . .

CMD [ "node","index.js" ]