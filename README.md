# Prometheus Native Histogram 

This repository serve as a demo for the new native histrogam from **Prometheus**.

For the reminder, native histogram is a feature that will allow creating more precise and performant histogram inside **Prometheus** than the current one (you know the series with the *le* label)

This repository will also demonstrate visualizing such histogram inside **Grafana**

## How to use this 

> You need docker and docker compose for this demonstration

Just : 
``` bash
docker compose up -d
```

Then you can visit prometheus here http://localhost:3000 and run promql query against the new histogram :
``` promql
rpc_durations_native_histogram_seconds
# or
rate(rpc_durations_native_histogram_seconds[30s])
```

Or visit this grafana dashboard to get a sneak peek at the heatmap for this new histogram compared to the old one http://localhost:3000/d/native-histogram