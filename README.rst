PowerMax Exporter
==================

About
------

Prometheus exporter for Dell PowerMax.

Usage
-------------

The exporter is able to expose metrics for below targets:

- Array: array overall performance;
- Cache: cache partition performance;
- FE Port: FE port performance;
- BE Port: BE port performance;
- Storage Group: storage group performance.

**Notes**:

- For each supported target, a different exporter needs to be used if more than one targets need to be monitored. This behavior may be changed in the future.
- The REST API of Unisphere for PowerMax supports collecting performance stats every 5 x minutes. However, "no content" may be returned even when 5 x minutes are used as the interval. For safe, 10 x minutes scraping interval is recommended to guarantee performance data can always be gotten.

Run as a docker container
~~~~~~~~~~~~~~~~~~~~~~~~~~

::

  git clone https://github.com/kckecheng/powermax_exporter.git
  cd powermax_exporter
  cp config.sample.yml config.yml
  vim config.yml # Tune options
  docker build -t kckecheng/powermax_exporter .
  docker run -d -p 9100:9100 --name powermax_exporter kckecheng/powermax_exporter

Run from CLI
~~~~~~~~~~~~~~

::

  cd powermax_exporter
  go build
  cp config.sample.yml config.yml
  vim config.yml # Tune options
  ./powermax_exporter -config config.yml
