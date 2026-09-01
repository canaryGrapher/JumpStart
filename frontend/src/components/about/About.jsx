import { useEffect, useState } from "react";
import { GetAboutInfo, BrowserOpenURL } from "../../api";
import AboutIdentity from "./AboutIdentity";
import AboutDetails from "./AboutDetails";
import UpdateSettings from "../UpdateSettings";

// Settings > About. Replaces the old "Updates" tab: same update controls,
// now sitting under the branding and build facts they belong to.
export default function About({ onError }) {
  const [info, setInfo] = useState(null);

  useEffect(() => {
    GetAboutInfo()
      .then(setInfo)
      .catch((e) => onError && onError(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="prefs-section about">
      <AboutIdentity info={info} />
      <AboutDetails info={info} />

      <hr className="about-rule" />

      <UpdateSettings onError={onError} />

      <p className="about-legal">
        © {new Date().getFullYear()} {info?.vendor || "Workvar"}. All rights reserved.
      </p>

      {info?.productUrl && (
        <div className="prefs-actions about-footer-actions">
          <button className="btn" onClick={() => BrowserOpenURL(`${info.productUrl}/#/privacy`)}>
            Read privacy policy
          </button>
        </div>
      )}
    </div>
  );
}
