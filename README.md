# csi-connectors
A common library of connectors to use for "connecting" CSI Volumes to Nodes

## Goals
Provide a library for making connections of volumes on Linux systems using various
transports.

We inentionally avoid pulling in additional dependencies, and we intend to be stateless
and as such are not using receivers.
