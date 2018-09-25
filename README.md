# CSI Connectors
## Goals
Provide a basic, lightweight library for CSI Plugin Authors to leverage some of the common tasks like connecting
and disconnecting iscsi and fc devices to a node.

We inentionally avoid pulling in additional dependencies, and we intend to be stateless
and as such are not using receivers.

## Non Goals
  * A kubernetes specific library
  * A kubernetes provisioner

# Design Philosophy
The idea is to keep this as lightweight and generic as possible.  We intentionally avoid the use of any third party
libraries or packages in the library.  We don't have a vendor directory, because we attempt to rely only on the std
golang libs.  This may prove to not be ideal, and may be changed over time, but initially it's a worthwhile goal.

Note that we're not using glog, or logrus, that's sure to raise hackles for folks.  We probably don't need them, and
we may or may not even need logging in the lib at all.  For those that require logrus or glog however the simple
logging module included with this library could be swapped for something like logrus or glog.
