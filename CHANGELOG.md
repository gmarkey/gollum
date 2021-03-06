# 0.4.1

This is a patch / minor features release

#### Fixed

 * InfluxDB JSON and line protocol fixed
 * shared.WaitGroup.WaitFor with duration 0 falls back to shared.WaitGroup.Wait
 * proper io.EOF handling for shared.BufferedReader and shared.ByteStream
 * HTTP consumer now responds with 200 instead of 203
 * HTTP consumer properly handles EOF
 * Increased test coverage

#### New

 * Support for InfluxDB line protocol
 * New setting to enable/disable InfluxDB time based database names
 * Introduction of "fuses" (circuit breaker pattern)
 * Added HTTPs support for HTTP consumer
 * Added POST data support to HTTPRequest producer

# 0.4.0

This release includes several reliability fixes that prevent messages from being lost during shutdown.
During this process the startup/shutdown mechanics were changed which introduced a lot of breaking changes.
Also included are improvements on the file, socket and scribe producers.
Write performance may show a minor increase for some producers.

This release contains breaking changes over version 0.3.x.
Custom producers and config files may have to be adjusted.

#### Breaking changes

 * shared.RuntimeType renamed to TypeRegistry
 * core.StreamTypes renamed to StreamRegistry
 * ?ControlLoop callback parameters for command handling moved to callback members
 * ?ControlLoop renamed to ?Loop, where ? can be a combination of Control (handling of control messages), Message (handling of messages) or Ticker (handling of regular callbacks)
 * PluginControlStop is now splitted into PluginControlStopConsumer and PluginControlStopProducer to allow plugins that are producer and consumers.
 * Producer.Enqueue now takes care of dropping messages and accepts a timeout overwrite value
 * MessageBatch has been refactored to store messages instead of preformatted strings. This allows dropping messages from a batch.
 * Message.Drop has been removed, Message.Route can be used instead
 * The LoopBack consumer has been removed. Producers can now drop messages to any stream using DropToStream.
 * Stream plugins are now allowed to only bind to one stream
 * Renamed producer.HttpReq to producer.HTTPRequest
 * Renamed format.StreamMod to format.StreamRoute
 * For format.Envelope postfix and prefix configuration keys have been renamed to EnvelopePostifx and EnvelopePrefix
 * Base64Encode and Base64Decode formatter parameters have been renamed to "Base64*"
 * Removed the MessagesPerSecAvg metric
 * Two functions were added to the MessageSource interface to allow blocked/active state query
 * The low resolution timer has been removed

#### Fixed

 * Messages stored in channels or MessageBatches can now be flushed properly during shutdown
 * Several producers now properly block when their queue is full (messages could be lost before)
 * Producer control commands now have priority over processing messages
 * Switched to sarama trunk version to get the latest broker connection fixes
 * Fixed various message loss scenarios in file, kafka and http request producer
 * Kafka producer is now reconnecting upon every error (intermediate fix)
 * StreamRoute formatter now properly works when the separator is a space
 * File, Kafka and HTTPRequest plugins don't hava mandatory values anymore
 * Socket consumer can now reopen a dropped connection
 * Socket consumer can now change access rights on unix domain sockets
 * Socket consumer now closes non-udp connections upon any error
 * Socket consumer can now remove an existing UDS file with the same name if necessary
 * Socket consumer now uses proper connection timeouts
 * Socket consumer now sends special acks on error
 * All net.Dial commands were replaced with net.DialTimeout
 * The makfile now correctly includes the config folder
 * Thie file producer now behaves correctly when directory creation fails
 * Spinning loops are now more CPU friendly
 * Plugins can now be addressed by longer paths, too, e.g. "contrib.company.sth"
 * Log messages that appear during startup are now written to the set log producer, too
 * Fixed a problem where control messages could cause shutdown to get stucked
 * The Kafka producer has been rewritten for better error handling
 * The scribe producer now dynamically modifies the batch size on error
 * The metric server tries to reopen connection every 5 seconds
 * Float metrics are now properly rounded
 * Ticker functions are now restarted after the function is done, preventing double calls
 * No empty messages will be sent during shutdown

#### New

 * Added a new stream plugin to route messages to one or more other streams
 * The file producer can now delete old files upon rotate (pruning)
 * The file producer can now overwrite files and set file permissions
 * Added metrics for dropped, discarded, filtered and unroutable messages
 * Streams can now overwrite a producer's ChannelTimeoutMs setting (only for this stream)
 * Producers are now shut down in order based on DropStream dependencies
 * Messages now keep a one-step history of their StreamID
 * Added format.StreamRevert to go back to the last used stream (e.g. after a drop)
 * Added producer.Spooling that temporary stores messages to disk before trying them again (e.g. useful for disconnect scenarios)
 * Added a new formatter to prepend stream names
 * Added a new formatter to serialize messages
 * Added a new formatter to convert collectd to InfluxDB (0.8.x and 0.9.x)
 * It is now possible to add a custom string after the version number
 * Plugins compiled from the contrib folder are now listed in the version string
 * All producers can now define a filter applied before formatting
 * Added unittests to check all bundled producer, consumer, format, filter and stream for interface compatibility
 * Plugins can now be registered and queried by a string based ID via core.PluginRegistry
 * Added producer for InfluxDB data (0.8.x and 0.9.x)
 * Kafka, scribe and elastic search producers now have distinct metrics per topic/category/index
 * Version number is now added to the metrics as in "MMmmpp" (M)ajor (m)inor (p)atch
