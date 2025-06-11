import Image from "next/image";
import './style.css';
import {Header} from "./Components/Header";
import FirstBlock from "./Components/FirstBlock";
import SecondBlock from "./Components/SecondBlock";
import OurSponsors from "./Components/OurSponsors";
import FourBlock from "./Components/FourBlock";

export default function Home() {
  return (
    <div className="main_container--dark_theme">
      <Header/>
      <div className="header-line"></div>
      <FirstBlock/>
      <SecondBlock/>
      <OurSponsors/>
      <div className="middle-block--dark_theme"></div>
      <FourBlock/>
    </div>
  );
}
