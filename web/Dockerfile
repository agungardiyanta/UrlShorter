FROM node:20.11.0-alpine AS build


WORKDIR /app
COPY package.json package-lock.json ./
# Install
RUN npm install 
COPY . ./
# Build
RUN npm run build

FROM node:20.11.0-alpine AS final
WORKDIR /app

RUN npm install -g serve

COPY --from=build /app/dist /app/dist
CMD [ "serve", "dist", "-s", "--no-clipboard"]
EXPOSE 3000