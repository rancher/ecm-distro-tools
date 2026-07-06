FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    libcurl4-openssl-dev \
    libyaml-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN git clone --depth 1 https://github.com/briandowns/librattler && \
    cd librattler && \
    make && make install && \
    ldconfig

COPY . /app

RUN make

CMD ["bin/chart_compare"]
