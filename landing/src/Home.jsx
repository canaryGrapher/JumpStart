import { onMount } from "solid-js";
import Nav from "./components/Nav";
import Hero from "./components/Hero";
import Features from "./components/Features";
import WhatItDoes from "./components/WhatItDoes";
import Ship from "./components/Ship";
import AiBoard from "./components/AiBoard";
import Faq from "./components/Faq";
import Contribute from "./components/Contribute";
import Footer from "./components/Footer";
import { initAnimations } from "./animations";
import { loadLatestRelease } from "./downloads";
import { initEngagement } from "./engagement";

export default function Home() {
  onMount(() => {
    initAnimations();
    loadLatestRelease();
    // Scroll depth, section visibility, and time on page. Set up after the
    // sections exist in the DOM, since it observes them by id.
    initEngagement();
  });
  return (
    <>
      <Nav />
      <Hero />
      <Features />
      <WhatItDoes />
      <Ship />
      <AiBoard />
      <Faq />
      <Contribute />
      <Footer />
    </>
  );
}
