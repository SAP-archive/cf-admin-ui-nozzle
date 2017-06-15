## Admin-ui-nozzle

This repo contains a [Cloud Foundry](https://www.cloudfoundry.org/platform/) application with a Firehose Nozzle implementation that forwards data to any Firehose client (forwarding their own authentication) and filters
the date received from Firehose before sending it back to the client.

The nozzle will forward only *ContainerMetric* and *ValueMetric* (ValueMetric if not 'latency' or 'route_lookup_time').
The filters has been tailored to reduce load on the [admin-ui](https://github.com/cloudfoundry-incubator/admin-ui) application.

The project is to fill the need of filtering the useless data coming from Firehose that hits the admin-ui performance. When Firehose will support fine filtering options this application will not be necessary anymore.



## Getting started

### Prerequisites

You need the [admin-ui](https://github.com/cloudfoundry-incubator/admin-ui) deployed and running. It does not matter if the admin-ui is deployed on Cloud Foundry or not, as long as there is
network connectivity between the admin-ui and other Cloud Foundry applications.

No other SAP software is required to run this application. 

### Deployment

- create an application manifest `manifest.yml` file starting from the example using the configuration parameters as described in the following session
- push the application on Cloud Foundry
- you can now use the nozzle url instead of the default doppler endpoint in the admin-ui by setting the property `doppler_logging_endpoint_override` in the admin-ui configuration. [More info at the admin-ui project page](https://github.com/cloudfoundry-incubator/admin-ui/blob/master/README.md#administration-ui-configuration)


## Configuration

`DOPPLER_ENDPOINT: wss://doppler.cf.bosh-lite.com:443` Doppler endpoint of your CF deployment.

`DEBUG: true` print logs every 30s some statistics on the filtered messages.

`SKIP_SSL_VALIDATION: true` skip ssl validation on the CF doppler endpoint.

## How to get support - how to contribute

Please open an issue or a pull request

## License
The project is released with the [Apache2.0 license](https://github.com/SAP/cf-admin-ui-nozzle/blob/master/LICENSE)
