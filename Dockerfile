FROM golang:1.20 AS builder

WORKDIR /wordsplit
COPY . .
RUN go build -o wordsplit ./cmd/wordsplit

FROM busybox AS runtime

RUN mkdir -p /dist && \
    wget -O /dist/words_alpha.txt https://raw.githubusercontent.com/dwyl/english-words/master/words_alpha.txt
COPY --from=builder /wordsplit/wordsplit /dist/wordsplit
ENV WORDS_FILE=/dist/words_alpha.txt

ENTRYPOINT [ "/dist/wordsplit" ]