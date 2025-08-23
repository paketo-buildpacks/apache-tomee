# Paketo Buildpack for Apache Tomee

## Buildpack ID: `paketo-buildpacks/apache-tomee`
## Registry URLs: `docker.io/paketobuildpacks/apache-tomee`

The Apache Tomee Buildpack is a Cloud Native Buildpack that contributes Apache Tomee and Process Types for WARs.

## Behavior
This buildpack will participate all the following conditions are met

* `<APPLICATION_ROOT>/WEB-INF` exists
* `Main-Class` is NOT defined in the manifest
* `BP_JAVA_APP_SERVER` is set to `tomee`

The buildpack will do the following:

* Requests that a JRE be installed
* Contribute a Tomee instance to `$CATALINA_HOME`
* Contribute a Tomee instance to `$CATALINA_BASE`
  * Contribute `context.xml`, `logging.properties`, `server.xml`, and `web.xml` to `conf/`
  * Contribute [Access Logging Support][als], [Lifecycle Support][lcs], and [Logging Support][lgs]
  * Contribute external configuration if available
* Contributes `tomee`, `task`, and `web` process types

### Tiny Stack

When this buildpack runs on the [Tiny stack](https://paketo.io/docs/concepts/stacks/#tiny), which has no shell, the following notes apply:
* As there is no shell, the `catalina.sh` script cannot be used to start Tomee
* The Tomee Buildpack will generate a start command directly. It does not support all the functionality in `catalina.sh`.
* Some configuration options such as `bin/setenv.sh` and setting `CATALINA_*` environment variables, will not be available.
* Tomee will be run with `umask` set to `0022` instead of the `catalina.sh`provided default of `0027`

[als]: https://github.com/cloudfoundry/java-buildpack-support/tree/master/tomcat-access-logging-support
[lcs]: https://github.com/cloudfoundry/java-buildpack-support/tree/master/tomcat-lifecycle-support
[lgs]: https://github.com/cloudfoundry/java-buildpack-support/tree/master/tomcat-logging-support

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_JAVA_APP_SERVER` | The application server to use. It defaults to the empty string, which allow the first Java application server in the buildpack order group to run. It can be set to `tomee` to force Tomee to be used. See the documentation for other Java application server buildpacks for other acceptable values. |
| `$BP_TOMEE_CONTEXT_PATH` | The context path to mount the application at.  Defaults to empty (`ROOT`).
| `$BP_TOMEE_ENV_PROPERTY_SOURCE_DISABLED` | When true the buildpack will not configure `org.apache.tomcat.util.digester.EnvironmentPropertySource`. This configuration option is added to support loading configuration from environment variables and referencing them in Tomcat configuration files.
| `$BP_TOMEE_EXT_CONF_SHA256` | The SHA256 hash of the external configuration package
| `$BP_TOMEE_EXT_CONF_STRIP` | The number of directory levels to strip from the external configuration package.  Defaults to `0`.
| `$BP_TOMEE_EXT_CONF_URI` | The download URI of the external configuration package
| `$BP_TOMEE_EXT_CONF_VERSION` | The version of the external configuration package
| `$BP_TOMEE_VERSION` |  Configure a specific Tomee version.  This value must _exactly_ match a version available in the buildpack so typically it would configured to a wildcard such as `9.*`.
| `$BP_TOMEE_DISTRIBUTION` |  Configure a specific Tomee distribution.  This value must be one of `microprofile`, `webprofile`, `plus` or `plume`. Defaults to `microprofile`.
| `$BPL_TOMEE_ACCESS_LOGGING_ENABLED` | Whether access logging should be activated.  Defaults to inactive.
| `BPI_TOMCAT_ADDITIONAL_JARS`              | This should only be used in other buildpacks to include a `jar` to the tomcat classpath. Several `jars` must be separated by `:`. |
| `BPI_TOMCAT_ADDITIONAL_COMMON_JARS`       | This should be used by other buildpacks to include additional locations to be class loaded by the tomcat common classloader. For example a buildpack might contribute resources in its dedicated layer and add the location with this variable to be classloaded additionally by Tomcat. Both folder paths as well as single `jar` file paths can be specified. |

### External Configuration Package
The artifacts that the repository provides must be in TAR format and must follow the Tomee archive structure:

```
<CATALINA_BASE>
└── conf
    ├── context.xml
    ├── server.xml
    ├── web.xml
    ├── ...
```

### Environment Property Source
When the Environment Property Source is configured, configuration for Tomcats [configuration files](https://tomcat.apache.org/tomcat-9.0-doc/config/systemprops.html) can be loaded
from environment variables. To use this feature, the name of the environment variable must match the name of the property.

## Bindings
The buildpack optionally accepts the following bindings:

## Providing Additional JARs to Tomcat

Buildpacks can contribute JARs to the `CLASSPATH` of Tomcat by appending a path to `BPI_TOMCAT_ADDITIONAL_JARS`.

```go
func (s) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	// Copy dependency into the layer
	file := filepath.Join(layer.Path, filepath.Base(s.Dependency.URI))

	layer, err := s.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		if err := sherpa.CopyFile(artifact, file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy artifact to %s\n%w", file, err)
		}
		return layer, nil
	})

	additionalJars := []string{file}
  // Add dependency to BPI_TOMCAT_ADDITIONAL_JARS
	layer.LaunchEnvironment.Append("BPI_TOMCAT_ADDITIONAL_JARS", ":", strings.Join(additionalJars, ":"))
	return layer, nil
}
```

### Type: `dependency-mapping`
|Key                   | Value   | Description
|----------------------|---------|------------
|`<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>`

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0

