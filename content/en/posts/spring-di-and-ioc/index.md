---
author: Nikita Zinovev
title: Why Spring Became Popular — Automating DI and IoC in Practice
date: 2025-08-15 09:15:00 +0300
description: How Spring made DI and IoC mainstream and convenient for everyday Java development
---

> **Revised Version**
> This is an updated version of my old article, originally published on January 14, 2024.
> Errors have been corrected, inaccuracies removed, redundant examples deleted, and the entire text reformatted for
> better readability.

One of the reasons why the Spring Framework gained massive popularity in the early 2000s was its ability to **automate
the application of the Inversion of Control (IoC) and Dependency Injection (DI) principles**.

These principles existed before Spring but were implemented manually, which required a lot of boilerplate code and discipline.
Spring introduced a container that creates objects (beans), configures them, and wires them together automatically, 
freeing developers from repetitive work.

## How It Was Before Spring

In the early Java EE (J2EE) era, DI had to be done manually. Objects often created their own dependencies, leading to strong coupling.

```java
public class OrderService {
    private InventoryService inventoryService = new InventoryServiceImpl();

    public void processOrder(Order order) {
        if (inventoryService.isAvailable(order.getItem())) {
            // process the order
        }
    }
}
```

This design created a tight dependency on the specific implementation `InventoryServiceImpl`. 
It made testing harder because you couldn't simply substitute a mock. If you needed to switch implementations, 
you had to modify the class code.

## "Manual" DI (Without Spring Yet)

Experienced teams were already practicing IoC/DI back then — they just did it manually:

```java
public class OrderService {
    private final InventoryService inventoryService;

    public OrderService(InventoryService inventoryService) {
        this.inventoryService = inventoryService;
    }

    public void processOrder(Order order) {
        if (inventoryService.isAvailable(order.getItem())) {
            // process the order
        }
    }
}
```

Now the dependency is passed **from the outside**, making it easy to substitute any implementation.

The downside of this approach was that in large projects, you had to manually create and "wire" dozens
or even hundreds of objects, which often led to mistakes during development.

## How It Became With Spring

Spring introduced an IoC container that:

- create beans (objects) automatically
- manages their lifecycle
- injects dependencies automatically

```java
@Service
public class OrderService {
    private final InventoryService inventoryService;

    @Autowired
    public OrderService(InventoryService inventoryService) {
        this.inventoryService = inventoryService;
    }

    public void processOrder(Order order) {
        if (inventoryService.isAvailable(order.getItem())) {
            // process the order
        }
    }
}
```

**What changed:**

- No need to manually create `InvoiceService` and pass it into `OrderService`
- At application startup, Spring finds all `InventoryService` implementations and injects the required one automatically.
- To change the implementation, you just register a new bean — no need to modify `OrderService` code.  

## Why It Took Off in the 2000s

By automating DI and IoC, Spring achieved the following:

- Scalability — the container creates and wires hundreds of objects automatically.
- Flexibility — easily swap implementations via configuration instead of rewriting code.
- Reduced boilerplate — less manual dependency setup code.
- Gradual adoption — you could use Spring solely for IoC/DI without rewriting the entire project.

DI and IoC were known before Spring, but Spring made them **mainstream** and convenient for everyday Java development,
eliminating routine work and minimizing human error in dependency setup.