-- +goose Up
-- The Exposure scope-act panel filters on action, and act carried no index that leads on it (#2073).
--
-- ListActsOfClassSince reads `action = @action AND created_at >= @from_time ORDER BY created_at DESC,
-- id DESC LIMIT @max_acts`, once per class in addressScopeActClasses, on every admin Exposure load.
-- act_created_at_idx leads on created_at, so the planner walks it backwards and filters action per
-- row. An estate that declared no seed this week matches nothing, so each of those reads walks every
-- act row inside the window to return none. 61 act classes write into that window and the panel reads
-- five of them, so the reads scale with act volume that has nothing to do with scopes.
--
-- Leading on action makes each read an equality seek, and created_at DESC, id DESC then supplies the
-- ORDER BY inside the group, so the LIMIT stops the scan rather than a sort consuming the window.
--
-- A partial index over the classes the panel names was weighed and rejected: it needs a migration per
-- class the reader gains, and the set has already moved once (#2169) before the first index existed.
--
-- act_created_at_idx stays. ListActsInRange carries no action predicate, so an index that leads on
-- action cannot serve it: this is an addition, not a replacement.
CREATE INDEX act_action_created_at_idx ON act (action, created_at DESC, id DESC);

-- +goose Down
DROP INDEX act_action_created_at_idx;
