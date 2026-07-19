import { useEffect, useState } from "react";
import {
  DockerInfo,
  ComposeUp,
  ComposeDown,
  ListContainers,
  ListImages,
  ListVolumes,
} from "../../api";
import ContainerList from "./ContainerList";
import ImageList from "./ImageList";
import VolumeList from "./VolumeList";

export default function ContainersPanel({ projectRoot, onError, onInfo }) {
  const [info, setInfo] = useState(null);
  const [containers, setContainers] = useState([]);
  const [images, setImages] = useState([]);
  const [volumes, setVolumes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [sub, setSub] = useState("containers");

  const loadResources = async () => {
    const [c, im, v] = await Promise.all([
      ListContainers(projectRoot).catch(() => []),
      ListImages(projectRoot).catch(() => []),
      ListVolumes(projectRoot).catch(() => []),
    ]);
    setContainers(c || []);
    setImages(im || []);
    setVolumes(v || []);
  };

  const load = async () => {
    setLoading(true);
    try {
      const dInfo = await DockerInfo(projectRoot);
      setInfo(dInfo);
      if (dInfo.available && dInfo.hasCompose) await loadResources();
    } catch (e) {
      onError(String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectRoot]);

  const run = async (label, action, successMsg) => {
    if (busy) return;
    setBusy(true);
    try {
      await action();
      onInfo && onInfo(successMsg || label);
      await loadResources();
    } catch (e) {
      onError(`${label} failed: ${String(e)}`);
    } finally {
      setBusy(false);
    }
  };

  if (loading && !info) {
    return (
      <div className="panel">
        <h3>Containers</h3>
        <div className="sub">Inspecting Docker…</div>
      </div>
    );
  }

  if (info && !info.available) {
    return (
      <div className="panel">
        <h3>Containers</h3>
        <div className="sub">
          Docker isn't running or the CLI isn't installed. Start Docker to manage
          this project's containers.
        </div>
        <div className="actions">
          <button className="btn small" onClick={load}>Retry</button>
        </div>
      </div>
    );
  }

  const counts = { containers: containers.length, images: images.length, volumes: volumes.length };

  return (
    <div className="panel dk-panel">
      <div className="git-panel-head">
        <h3>Containers</h3>
        {info && info.composeFile && (
          <span className="kb-pill" title="Compose file">{info.composeFile}</span>
        )}
      </div>

      {info && !info.hasCompose ? (
        <div className="sub">
          A Dockerfile was found but no compose file. Add a docker-compose file to
          bring a stack up and down from here.
        </div>
      ) : (
        <>
          <div className="actions">
            <button className="btn primary" disabled={busy} onClick={() => run("Compose up", () => ComposeUp(projectRoot), "Stack started")}>
              {busy ? "Working…" : "Up"}
            </button>
            <button className="btn danger" disabled={busy} onClick={() => run("Compose down", () => ComposeDown(projectRoot), "Stack stopped")}>
              Down
            </button>
            <button className="btn small" disabled={busy} onClick={loadResources}>
              Refresh
            </button>
          </div>

          <div className="dk-subtabs">
            {["containers", "images", "volumes"].map((k) => (
              <button
                key={k}
                className={sub === k ? "active" : ""}
                onClick={() => setSub(k)}
              >
                {k[0].toUpperCase() + k.slice(1)} <span className="dk-count">{counts[k]}</span>
              </button>
            ))}
          </div>

          {sub === "containers" && (
            <ContainerList containers={containers} busy={busy} onAction={run} />
          )}
          {sub === "images" && <ImageList images={images} />}
          {sub === "volumes" && <VolumeList volumes={volumes} />}
        </>
      )}
    </div>
  );
}
