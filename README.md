
## What is it?
This is a utility with a set of tests to explore and compare different ways of detecting ‘memory pressure’ (the system is running out of free memory) state in Linux kernel-based OS.

It consists of a few sub-components, that collect and analyze various metrics:

### /proc/meminfo observer
This reads and parses ```/proc/meminfo``` file and generates the following metrics:
```mem_total``` - total system physical memory (in megabytes)
```mem_avail_est``` -  the amount of memory that is available for a new workload, without pushing the system into swap, estimated from `MemFree`, `Active(file)`, `Inactive(file)`, and `SReclaimable`, as well as the "low" watermarks from `/proc/zoneinfo` file. This is counted using a legacy algorithm from Linux kernel, see https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=34e431b0a patch description and code comments for the details and explanations.
```mem_avail``` - the amount of memory that is available for a new workload, without pushing the system into swap, ```'MemAvailable'``` param from ```/proc/meminfo``` file. Current Linux kernel versions count this in a slightly different way than in a previous metric.
```mem_pcnt``` - percent of physical memory filling. This can be used for "simple" memory pressure threshold-based evaluation.
```swp_pcnt``` - percent of swap usage
```swp_free``` - free swap space (in megabytes)
```swp_total``` - total swap spaces  (in megabytes)

### Page faults counter and low watermark pages observer
One task of this observer is to monitor ```'pgmajfault'``` (page faults counter) parameter. In case if current faults per second value is significantly higher than the average, we can assume that swap trashing is happening. Because sample times are inconsistent and we're measuring CPU time instead of real time, EWMA low-pass filter is applied for the values. 
Another task is to count 'swap tendency' metric, as it is described here https://access.redhat.com/solutions/103833
The actual problem is that 'swappines' is a standard system param, 'mapped ratio' can be counted from 'nr_mapped' system metric, but the 'distress' value is inaccessible from the kernel internal for user-space software, so this method, unfortunately, is unusable in production.
Metrics:
```swp_flts_sec``` - page faults per second
```swp_flts_sec_f``` - page faults per second with EWMA low-pass filter
```swp_flts_mult```- current page faults per second and average page faults per second ratio. This can be used for swap trashing evaluation.
```swp_tend``` - 'swap tendency' metric counted as described above.


### cgroups eventfd observer
This set up cgroups 'memory_pressure' event file descriptor and subscribes for these events. CGroups subsystem allows us to set physical and virtual memory limits for the process or process group. "Memory pressure" eval is based on "scanned/reclaimed pages" ratio, see Linux kernel comments for details:
https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/mm/vmpressure.c?id=34e431b0ae398fc54ea69ff85ec700722c9da773
```/sys/fs/cgroup/memory/cgroup.event_control``` is used to subscribe for events from ```/sys/fs/cgroup/memory/memory.pressure_level``` - this is a standard cgroups mechanism.
Metric from this observer:
```cgroups``` - bit mask ('critical - 'medium' - 'low'). E.g., in case of 'low' trigger, the value will be 1, in case of all triggers active, the value will be equal to 7.
This observer may require superuser rights to initialize and run.

### PSI (pressure stall information) observer
PSI aggregates and reports the overall wallclock time in which the
tasks in a system wait for contended hardware resources. In modern Linux kernels, ```/proc/pressure/memory``` file provides information on the time that processes spend waiting due to memory pressure.
The metics are:
```psi_some``` - the percentage of the time that at least one process could be running if it weren't waiting for memory resources
```psi_full``` - the percentage of time that nobody is able to use the CPU for actual work due to memory pressure
In both cases, ```avg10``` (10 seconds averaged) values are used.

### Allocator
Allocator is used for allocating (^^_^^) new memory block every N (N = 5) seconds. Because  'overcommit memory' feature is enabled by default on modern Linux systems, allocator also fills one byte in every memory page with a random value to force the system memory allocator to allocate the memory page (TODO: rewrite this paragraph in a human-readable style :) )
The default block size is 128 megabytes, it can also be specified as a first command-line argument.
In case if 'block size' value is set to 0, 'memory-pressure' binary will not initialize and start the allocator module at all and will work simply in passive mode.
Allocator reports total allocated memory blocks size to ```'alloctd'``` metic.

### Tracker
It collects all the metrics from the sub-components and prints to the stdout every N (N=5, right) seconds or in case of events. It shows adds the time from the process start (in seconds) in the ```'time'``` metric.
It looks like this:
<pre>2019/09/18 19:58:01 System page size is 4096 bytes
2019/09/18 19:58:01 System timer frequency is 100 Hz
2019/09/18 19:58:01 Using block size 128 Mb
alloctd, cgroups, mem_avail, mem_avail_est, mem_pcnt, mem_total, psi_full, psi_some, swp_flts_mult, swp_flts_sec, swp_flts_sec_f, swp_free, swp_pcnt, swp_tend, swp_total, time, 
    640,       0,  21502.88,      21524.23,    32.99,  32091.43,     0.00,     0.00,          0.88,         0.00,           9.81, 30518.00,     0.00,    11.75,  30518.00,    5, 
   1280,       0,  20837.51,      20858.86,    35.07,  32091.43,     0.00,     0.00,          0.79,         0.61,           8.83, 30518.00,     0.00,    11.75,  30518.00,   10, 
   1920,       0,  20172.38,      20193.73,    37.14,  32091.43,     0.00,     0.00,          0.70,         0.00,           7.83, 30518.00,     0.00,    11.75,  30518.00,   15, 
   2560,       0,  19509.39,      19530.74,    39.21,  32091.43,     0.00,     0.00,          0.63,         0.00,           7.11, 30518.00,     0.00,    11.75,  30518.00,   20, 
   2944,       1,  18992.82,      19002.17,    40.82,  32091.43,     0.00,     0.00,          0.59,         0.00,           6.55, 30518.00,     0.00,    11.75,  30518.00,   23, 
</pre>


## How to run and test
```git clone```, ```go build``` and run!
And, of course, ```go test``` if you need this.

