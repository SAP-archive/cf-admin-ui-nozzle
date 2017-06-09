## Admin-ui-nozzle

This repo contains a [Cloud Foundry](https://www.cloudfoundry.org/platform/) application with a Firehose Nozzle implementation that forwards data to any Firehose client (forwarding their own authentication) and filters
the date received from Firehose before sending it back to the client.

The nozzle will forward only *ContainerMetric* and *ValueMetric* (ValueMetric if not 'latency' or 'route_lookup_time').
The filters has been tailored to reduce load on the [admin-ui](https://github.com/cloudfoundry-incubator/admin-ui) application.

The project is to fill the need of filtering the useless data coming from Firehose that hits the admin-ui performance. When Firehose will support fine filtering options this application will not be necessary anymore.

No extra SAP software is required to run this application. You just need to have administrator access to a Cloud Foundry installation and the [admin-ui](https://github.com/cloudfoundry-incubator/admin-ui) already deployed as an application. 

## Getting started
- clone the repository
- create a `manifest.yml` file following the example with your configuration settings
- push the application on cloud foundry in the same space where the admin-ui is deployed
- you can now use the nozzle url instead of the default doppler endpoint in the admin-ui by setting the property `doppler_logging_endpoint_override` in the admin-ui configuration. [More info at the admin-ui project page](https://github.com/cloudfoundry-incubator/admin-ui/blob/master/README.md#administration-ui-configuration)


## Configuration

`DOPPLER_ENDPOINT: wss://doppler.cf.bosh-lite.com:443` Doppler endpoint of your CF deployment.

`DEBUG: true` print logs every 30s some statistics on the filtered messages.

`SKIP_SSL_VALIDATION: true` skip ssl validation on the CF doppler endpoint.

## How to get support - how to contribute

Please open an issue or a pull request

## License
The project is released with the [Apache2.0](https://www.apache.org/licenses/LICENSE-2.0) license
