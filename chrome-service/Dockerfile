FROM node:20-alpine

COPY package.json package.json
COPY package-lock.json package-lock.json
COPY index.js index.js


RUN npm install

EXPOSE 3000

CMD [ "node", "index.js" ]
