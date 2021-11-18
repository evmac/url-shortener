`make all` to initialize project.
This will build the containers, instantiate dependencies, and run the tests.

Notes:
- I have included Postman collections to facilitate easy API testing and to showcase the functionality.
- I did not build a frontend as it would have limited value for this iteration.
- I did not productionize the Docker containers - this is purely a development project.
This could be further improved by multi-stage builds and a separate test Dockerfile.
- I would likely leverage some community projects where I to attempt to build this again.
Working with the low-level Golang packages is speedy, but anything larger than this needs a more robust solution for route-handling and testing purposes.

There are two components to this solution: the main URL shortening app and a key generation service.

The URL shortening app is backed by Elasticsearch for quick retrieval of already generated short URLs.
It supports both internal and external redirects for hosts to allow for use of our default hostname as well as customized short hostnames.
Further iterations could support deleting and modifying existing URLs.

The key generation service is backed by Postgres to track URL sources and the keys associated.
This enables key uniqueness among any number of sources.
Separating the key generation from URL shortening allows us to scale each independently.
It also allows us to trial different methods of key generation.
I sourced a cryptographically-secure solution as it allowed us to generate URL-valid keys of any length with guaranteed uniqueness.

Coverage:
- urlshortenapp: 88.4% of statements
- keygensvc: 73.9% of statements

Coverage is lower on keygensvc as low-level Postgres interaction is not tested at all.
Mocking out the DB connection didn't seem to provide enough value to make it worthwhile, but this can be improved if more functionality is required.
Additionally, neither app instantiation is unit-tested.
There's limited value to this, and any failures at this level can be easily identified in development.
Overall, I wouldn't call these tests "complete", but they do cover the majority of functionality.