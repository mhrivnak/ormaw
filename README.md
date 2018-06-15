# ormaw
Owner Reference Mutating Admission Webhook

This is a proof-of-concept webhook for use with Kubernetes 1.10+.

## Purpose

In some scenarios, creation of an application's resources can be delegated to
third-party assets such as Ansible roles or Helm charts.  It is desirable to
ensure that each created resource has a consistent Owner Reference, such as a
reference to a Custom Resource associated with an Operator.

## How it Works

A new Service Account should be created for each application, and all resource
creation must be done with that account. The service account should have its
owner reference set to a Custom Resource that is the same CR that represents
the application.

When the webhook sees that a resource is being created, it will check the
service account for an owner reference of the correct type, and if found, add
it to the resource being created.

## Deployment

* create a CA and a cert/key pair
* base64-encode the PEM-encoded CA and add it to the webhook.yaml.example
* put the cert/key pair in the right place as hard-coded in main.go
* build and run the service, setting the `CRD` environment variable to the Kind of your CR
* create the webhook from webhook.yaml.example
* create a CRD and a CR from it
* create a service account with an owner reference to the CR
* create some other resource using the service account, and it should automatically get the owner reference added
