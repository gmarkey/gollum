Null
====

This producers discards all messages similar to a /dev/null.
Its main purpose is to profile consumers and/or streams.
This producer does not implement a fuse breaker.
See the `API documentation <http://gollum.readthedocs.org/en/latest/producers/null.html>`_ for additional details.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**ID**
  Allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.
**Stream**
  Defines either one or an aray of stream names this producer receives messages from.

Example
-------

.. code-block:: yaml

  - "producer.Null":
    Enable: true
    Stream:
        - "log"
        - "console"
