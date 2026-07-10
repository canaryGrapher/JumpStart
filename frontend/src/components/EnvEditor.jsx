export default function EnvEditor({ env, onChange }) {
  const entries = Object.entries(env);

  const update = (list) => onChange(Object.fromEntries(list.filter(([k]) => k)));

  const setKey = (i, key) => {
    const list = [...entries];
    list[i] = [key, list[i][1]];
    update(list);
  };

  const setVal = (i, val) => {
    const list = [...entries];
    list[i] = [list[i][0], val];
    update(list);
  };

  const remove = (i) => update(entries.filter((_, idx) => idx !== i));

  return (
    <div>
      {entries.map(([k, v], i) => (
        <div className="env-row" key={i}>
          <input value={k} placeholder="KEY" onChange={(e) => setKey(i, e.target.value)} />
          <input value={v} placeholder="value" onChange={(e) => setVal(i, e.target.value)} />
          <button className="link-btn" onClick={() => remove(i)}>
            x
          </button>
        </div>
      ))}
      <button
        className="link-btn"
        onClick={() => onChange({ ...env, ["VAR_" + (entries.length + 1)]: "" })}
      >
        + Add variable
      </button>
    </div>
  );
}
