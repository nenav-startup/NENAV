# ADR 0003: PostGIS Spatial Data Engine for Geospatial Discovery

## Status
Accepted

## Context
NENAV is an on-demand marketplace connecting customers with verified local professionals based on dynamic real-time proximity, operating radiuses, and service polygons.

## Decision
We adopt **PostGIS** on top of PostgreSQL 16:
- Professional service territories and customer booking locations will be stored as spatial geometries (`geography(Point, 4326)`).
- Geospatial queries will leverage native GiST spatial indexes.
- Distance calculations and radius matching will execute in-database via `ST_DWithin` and `ST_DistanceSphere`.

## Consequences
- High-efficiency geospatial filtering without adding external geosearch engines (Elasticsearch/OpenSearch).
- Single datastore for relational integrity, spatial indexes, and ACID guarantees.
