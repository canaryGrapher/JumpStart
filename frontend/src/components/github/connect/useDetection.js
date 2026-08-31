import { useCallback, useEffect, useState } from "react";
import { GitHubDetectRepo } from "../../../api";

// How long the "figuring things out" state stays on screen at minimum.
// Real detection is usually near-instant (local git status + one cached
// GitHub lookup), which would otherwise make the checklist flash by
// before anyone can read it.
const MIN_VISIBLE_MS = 900;

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

// Runs GitHubDetectRepo and holds the result until at least
// MIN_VISIBLE_MS has passed, so the detecting animation always gets a
// beat to play out. Exposes a `refresh` so a step that changes the
// project's GitHub state (connecting, creating a repo) can re-run the
// scan without the caller managing its own loading state.
export default function useDetection(projectId) {
  const [detection, setDetection] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    const startedAt = Date.now();
    try {
      const result = await GitHubDetectRepo(projectId);
      const elapsed = Date.now() - startedAt;
      if (elapsed < MIN_VISIBLE_MS) await sleep(MIN_VISIBLE_MS - elapsed);
      setDetection(result);
      return result;
    } catch (e) {
      setError(String(e));
      return null;
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  return { detection, error, loading, refresh };
}
