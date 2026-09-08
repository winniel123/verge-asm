# 1756: Inventory rows for 162.222.48.0/22 on this dev machine

Queried 2026-09-08T22:46:22Z against compose service verge-asm-postgres-1 (volume verge-asm_pgdata, created 2026-09-06T20:25:32Z).

```
-- step 1: report evidence
select count(*) rows, count(distinct split_part(subject_key,':',1)) addrs, count(*) filter (where closed_at is null) open_rows, count(*) filter (where value->>'outcome'='reached') reached_rows, min(opened_at), max(opened_at) from span where subject_kind='service' and facet='reachability' and (subject_key like '162.222.48.%' or subject_key like '162.222.49.%' or subject_key like '162.222.50.%' or subject_key like '162.222.51.%');
rows$'\t'addrs$'\t'open_rows$'\t'reached_rows$'\t'min$'\t'max
0$'\t'0$'\t'0$'\t'0$'\t'$'\t'
(1 row)

-- any mention of 162.222. in span or observation
t$'\t'count
span$'\t'0
observation$'\t'0
(2 rows)

-- what the database holds instead
seeds$'\t'batches$'\t'observations$'\t'spans$'\t'first_span_created
0$'\t'0$'\t'0$'\t'17$'\t'2026-09-06 20:25:43.15677+00
(1 row)

-- every span row
id$'\t'subject_kind$'\t'facet$'\t'subject_key$'\t'outcome$'\t'is_gap$'\t'vantage_id$'\t'opened_at$'\t'opened_batch_id$'\t'created_at
1$'\t'name$'\t'resolution$'\t'www.acmecorp.io$'\t'$'\t'f$'\t'$'\t'2026-07-14 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
2$'\t'name$'\t'dns-record$'\t'www.acmecorp.io$'\t'$'\t'f$'\t'$'\t'2026-07-14 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
3$'\t'name$'\t'resolution$'\t'api.acmecorp.io$'\t'$'\t'f$'\t'$'\t'2026-06-02 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
4$'\t'name$'\t'dns-record$'\t'api.acmecorp.io$'\t'$'\t'f$'\t'$'\t'2026-06-02 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
5$'\t'name$'\t'resolution$'\t'mail.acmecorp.io$'\t'$'\t'f$'\t'$'\t'2026-05-19 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
6$'\t'name$'\t'dns-record$'\t'mail.acmecorp.io$'\t'$'\t't$'\t'$'\t'2026-08-21 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
7$'\t'service$'\t'tls-acceptance$'\t'198.51.100.7:443/tcp$'\t'enumerated$'\t'f$'\t'$'\t'2026-07-14 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
8$'\t'service$'\t'certificate$'\t'198.51.100.7:443/tcp$'\t'$'\t'f$'\t'$'\t'2026-08-03 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
9$'\t'service$'\t'reachability$'\t'203.0.113.44:22/tcp$'\t'answers$'\t'f$'\t'$'\t'2026-04-30 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
10$'\t'service$'\t'tls-acceptance$'\t'203.0.113.44:22/tcp$'\t'none · plaintext ssh$'\t'f$'\t'$'\t'2026-04-30 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
11$'\t'service$'\t'certificate$'\t'198.51.100.31:8443/tcp$'\t'$'\t't$'\t'$'\t'2026-08-19 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
12$'\t'endpoint$'\t'http-identity$'\t'www.acmecorp.io · :443 https$'\t'$'\t'f$'\t'$'\t'2026-07-14 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
13$'\t'endpoint$'\t'http-identity$'\t'grafana.acmecorp.io · :443 https$'\t'$'\t'f$'\t'$'\t'2026-06-27 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
14$'\t'address$'\t'reachability$'\t'198.51.100.7$'\t'answers$'\t'f$'\t'$'\t'2026-07-14 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
15$'\t'address$'\t'reachability$'\t'198.51.100.7$'\t'answers$'\t'f$'\t'$'\t'2026-08-02 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
16$'\t'address$'\t'reachability$'\t'203.0.113.44$'\t'answers$'\t'f$'\t'$'\t'2026-04-30 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
17$'\t'address$'\t'reachability$'\t'104.18.22.90$'\t'gap$'\t't$'\t'$'\t'2026-08-19 00:00:00+00$'\t'$'\t'2026-09-06 20:25:43.15677+00
(17 rows)
```
