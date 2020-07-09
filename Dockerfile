FROM scratch

COPY bin/gnockgnock .

ENV HOST=0.0.0.0
ENV PORT=8080
ENV LOG_LEVEL=info

EXPOSE ${PORT}

CMD ["./gnockgnock"]