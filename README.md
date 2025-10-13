# Sophons

A fast, Ansible-compatible, configuration management tool.

## Status

This is **extremely alpha state**. At time of writing only file operations are
supported to some extent (some attributes are not implemented) with the most
basic support.

The following still need implementation (in order of priority, not an exhaustive
list):
  * all Ansible builtins
  * better roles support (files, templates, handlers, etc)
  * secure execution
  * better SSH support (password auth, reading SSH config, etc)
  * collections
  * extensibility
  * agent mode (long-running vs current implementation that's short-lived)

### Speed

Because of its implementation, Sophons is **orders of magnitude** faster than
Ansible. A simple benchmark of creating 1000 file one after the other against a
machine on the same LAN has Ansible clocking in at **500 seconds** of execution
time. Sophons does as much in **2 seconds**.

Of course this "benchmark" is extremely simplistic and does not leverage
parallel execution, nor is it fully representative. But it demonstrates well why
Sophons can be much much faster than Ansible.

### Security

At time of writing, the execution model means every host gets the full playbook
including potentially sensitive information. This is very obviously problematic
as an attacker could leverage that to retrieve e.g. credentials by simply adding
a node to the inventory. This will get addressed in time.

Do note that this project has not seen any review by a security professional and
thus **should not be used in production**.

### Compatibility

Currently compatibility with Ansible is on a best-effort basis. Some peculiar
behaviour of Ansible may be dropped in favor of a saner or faster
implementation.

Do note also that Windows support is not planned at time of writing.

## Usage

### Local Execution

Simply run the `executer` binary, passing in the playbook to run as argument:

```shell
executer playbook.yaml
```

### Remote Execution

The `dialer` binary expected pre-compiled binaries to be available ahead of time
to the target OSes and architecture, e.g.:

```shell
❯ ls bin/
executer-darwin-arm64  executer-darwin-x86_64  executer-linux-arm64  executer-linux-x86_64
```

Once available, the `dialer` can be run as follows:

```shell
dialer -b bin/ -i inventory.yaml playbook.yaml
```

Flags are available to provide an SSH username and private key. See `dialer -h`
for more information.

## Architecture

The main idea behind making Sophons fast is realising that Ansible's own
execution model is flawed: it copies a Python script to the controlled host for
every task and executes it. Instead Sophons is decomposed in two small
components:
* a dialer, that connects to controlled nodes, copies the inventory and playbook
  as well as the executer to them in a temporary directory, and then runs said
  executer for local execution,
* an executer, tasked with running the actual operations against the host
  without the network overhead for each task.

## Why

### Speed

Ansible is unbelievably slow for 2 main reasons:
* it relies on Python
* it relies on SSH

Unfortunately it's the combination of the two, and Ansible's design, that make
it so incredibly slow: Ansible needs to SSH to every node for every task,
and then execute Python code. If the controller node is hosted far from the
controlled nodes (a reasonable situation: imagine a Europe-based remote employee
at a large California-centric tech company), then each task will add significant
latency to its duration. On top of that starting Python itself for each task
will further increase the duration, magnified by the fact that Python code is
also not incredibly fast in the first place.

Those problems are solvable without ditching Python or SSH. Some prior art in
improving speed of execution include
[Mitogen](https://mitogen.networkgenomics.com/ansible_detailed.html). Similarly,
the Python overhead can be minimised. That being said doing so would require an
almost full rewrite of Ansible, which at time of writing looks extremely
unlikely.

### Evolution of Systems Design

Since the inception of the biggest configuration engines the likes of Ansible
(Puppet, SaltStack, Chef, etc), a lot has happened in improving how distributed
systems interact. Unfortunately no configuration engine that has taken
advantages of recent technologies has seen large-scale adoption.

This project aims to apply the last decade of learnings to what is essentially a
distributed systems problem. It seeks to build a reliable, fast, correct and
secure configuration engine, without compromising on compatibility or ease of
use.

### Naming

An Ansible is a device mention in Ursula K. Le Guin's novels that allows
faster-than-light communication, albeit with limited bandwidth. By comparison,
Sophons are devices from Liu Cixin's Three Body Problem novels, that allow
accessing any electronic device and communicate back instantaneously, i.e. a FTL
communication device with no bandwidth limitation. It felt fitting to pick a
device with better capabilities than the "plain" (though admittedly beyond
anything that presently exists) ansible.
