import { useEffect, useState } from "react";
import { ListEditors } from "../api";

// The installed-editor list doesn't change while JumpStart is running, and
// every OpenActions instance in the header/process cards wants the same
// list, so it's fetched once and shared the same way useAppIcon shares its
// icons.
let editorsPromise = null;

function sharedEditors() {
  if (!editorsPromise) {
    editorsPromise = ListEditors()
      .then((list) => list || [])
      .catch(() => []);
  }
  return editorsPromise;
}

// useEditors returns every code editor JumpStart found installed on this
// machine (each with its own real application icon), or [] until the list
// resolves or if none were found.
export default function useEditors() {
  const [editors, setEditors] = useState([]);

  useEffect(() => {
    let active = true;
    sharedEditors().then((list) => {
      if (active) setEditors(list);
    });
    return () => {
      active = false;
    };
  }, []);

  return editors;
}
