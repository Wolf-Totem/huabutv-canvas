import { useEffect, useState } from "react";
import { Link, useParams } from "react-router";

import { ossProcessedImage } from "@/lib/oss-image";
import { PLAZA_DEMO_AUTHOR, watchFromPlazaWork, type FeaturedCanvas } from "@/lib/plaza-catalog";
import { listPlazaWorks } from "@/services/api/plaza";
import "./plaza-watch.css";

export default function PlazaProfilePage() {
    const { userId = PLAZA_DEMO_AUTHOR.id } = useParams();
    const [works, setWorks] = useState<FeaturedCanvas[]>([]);
    useEffect(() => {
        let active = true;
        listPlazaWorks({ page: 1, pageSize: 80, sort: "hot" })
            .then((list) => {
                if (!active) return;
                const all = list.works.filter((work) => work.allowProcessView).map(watchFromPlazaWork);
                const mine = all.filter((item) => !userId || item.authorId === userId || userId === PLAZA_DEMO_AUTHOR.id || userId === "featured");
                setWorks(mine.length ? mine : all);
            })
            .catch(() => {
                if (active) setWorks([]);
            });
        return () => {
            active = false;
        };
    }, [userId]);
    const isDemo = !userId || userId === PLAZA_DEMO_AUTHOR.id || userId === "featured" || works.some((item) => item.authorId === userId);
    const name = isDemo ? PLAZA_DEMO_AUTHOR.name : userId;
    const likes = works.reduce((sum, item) => sum + (item.likeCount || 0), 0);
    const cover = ossProcessedImage(works[0]?.coverUrl, 1600) || works[0]?.coverUrl || "";
    return (
        <main className="plaza-profile">
            <div className="plaza-profile-top">
                <Link to="/create" className="plaza-profile-back">返回创作</Link>
            </div>
            <div className="plaza-profile-banner">{cover ? <img src={cover} alt="" /> : null}</div>
            <div className="plaza-profile-head">
                <img className="plaza-profile-avatar" src={PLAZA_DEMO_AUTHOR.avatarUrl} alt="" />
                <div className="plaza-profile-copy">
                    <h1>{name}</h1>
                    <p>{isDemo ? PLAZA_DEMO_AUTHOR.bio : "创作者主页"}</p>
                    <div className="plaza-profile-stats">
                        <span><strong>{works.length}</strong> 作品</span>
                        <span><strong>{likes}</strong> 获赞</span>
                    </div>
                </div>
            </div>
            <div className="plaza-profile-tabs">
                <span className="is-current">作品</span>
            </div>
            <div className="plaza-profile-grid">
                {works.map((item) => (
                    <Link key={item.slug} to={`/plaza/${item.slug}`} className="plaza-profile-card">
                        <span className="plaza-card-media">
                            <img src={ossProcessedImage(item.coverUrl, 720) || item.coverUrl} alt="" />
                        </span>
                        <strong>{item.title}</strong>
                    </Link>
                ))}
            </div>
        </main>
    );
}
